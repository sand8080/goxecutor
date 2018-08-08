package task

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// ID is human readable unique per graph task identifier
type ID string

// Status is task status
type Status string

const (
	// StatusNew status of new task
	StatusNew       Status = "NEW"
	// StatusWaiting status of task waiting for required tasks completion
	StatusWaiting   Status = "WAITING"
	// StatusRunning status of executing task
	StatusRunning   Status = "RUNNING"
	// StatusReady status of successfully executed task
	StatusReady     Status = "READY"
	// StatusCancelled status of cancelled task
	StatusCancelled Status = "CANCELLED"
	// StatusError status of task executed with error
	StatusError     Status = "ERROR"
)

// ErrNotChild error on adding a child task that not requires the task
var ErrNotChild = errors.New("adding non required child")

// ErrChildAlreadyAdded error on adding already added child
var ErrChildAlreadyAdded = errors.New("child already added")

// DoFunc is task payload processing function.
// If DoFunc returns not nil value and there are no error the value will be saved in context as value
// with task.ID as key.
type DoFunc func(ctx context.Context, payload interface{}) (interface{}, error)

// Result describes task execution result
type Result struct {
	id     ID
	result interface{}
}

// Task describes task structure
type Task struct {
	sync.Mutex
	// UUID is storage specific identifier
	UUID uuid.UUID
	// ID is human readable unique per graph task identifier
	ID ID
	// Status is task status
	Status Status
	// Requires is map of tasks required for the task execution
	Requires      map[ID]bool
	// Payload is task payload
	Payload       interface{}
	do            DoFunc
	// DoResult is result of task execution
	DoResult      []byte
	// DoError is error of task execution
	DoError       error
	undo          DoFunc
	// UndoResult is result of task revert
	UndoResult    []byte
	// UndoError is error of task revert
	UndoError     error
	// RequiredFor is map of tasks requires the task execution
	RequiredFor   map[ID]bool
	waitingResult chan Result
	notifyResult  []chan<- Result
}

// NewTask creates new task
func NewTask(id ID, requires []ID, payload interface{}, doFunc DoFunc, undoFunc DoFunc) *Task {
	reqSet := make(map[ID]bool, len(requires))
	for _, req := range requires {
		reqSet[req] = true
	}
	var waitingID chan Result
	if len(requires) > 0 {
		waitingID = make(chan Result, len(requires))
	}
	return &Task{
		ID:            id,
		Status:        StatusNew,
		Requires:      reqSet,
		RequiredFor:   make(map[ID]bool),
		Payload:       payload,
		do:            doFunc,
		undo:          undoFunc,
		waitingResult: waitingID,
	}
}

// AddChild adds child task.
func (t *Task) AddChild(child *Task) error {
	t.Lock()
	defer t.Unlock()

	log.Debugf("Adding child %q to %q", child.ID, t.ID)
	if !child.Requires[t.ID] {
		return ErrNotChild
	}

	if added, ok := t.RequiredFor[child.ID]; ok && added {
		return ErrChildAlreadyAdded
	}

	t.notifyResult = append(t.notifyResult, child.waitingResult)
	t.RequiredFor[child.ID] = true
	return nil
}

func waitingReadiness(ctx context.Context, cancelFunc context.CancelFunc, task *Task) context.Context {
	if task.waitingResult != nil {
		log.Debugf("Task %v is waiting for completion of required tasks", task.ID)
		task.Status = StatusWaiting
		received := make(map[ID]bool, len(task.Requires))

	loop:
		for {
			select {
			case res := <-task.waitingResult:
				log.Debugf("%v is notified: task %q is finished.", task.ID, res.id)
				received[res.id] = true

				if res.result != nil {
					log.Debugf("Adding task %v result into context", res.id)
					ctx = context.WithValue(ctx, res.id, res.result)
				}

				// Checking if all requirements are satisfied
				if len(received) == len(task.Requires) {
					log.Debugf("All required tasks completed for %v", task.ID)
					break loop
				}
			case <-ctx.Done():
				log.Infof("Cancelling task %q", task.ID)
				task.Status = StatusCancelled
				cancelFunc()
				break loop
			}
		}
	}
	return ctx
}

func prepareResult(result *interface{}) []byte {
	if *result == nil {
		return nil
	}
	resMarsh, errMarsh := json.Marshal(*result)
	if errMarsh != nil {
		log.Errorf("Task execution result marshalling error: %v. Result: %v", errMarsh, result)
	}
	return resMarsh
}

func saveDoResult(task *Task, result *interface{}) {
	task.DoResult = prepareResult(result)
}

// Do executes task
func Do(ctx context.Context, cancelFunc context.CancelFunc, task *Task, storage Storage) error {
	// Task lock prevents events mess up in case of multiple Do calls with the same task object.
	task.Lock()
	defer task.Unlock()
	defer func() {
		if err := storage.Save(task); err != nil {
			log.Errorf("Saving task %v(%v) error: %v", task.ID, task.UUID, err)
		}
	}()

	log.Debugf("Execution of task %q initiated", task.ID)

	ctx = waitingReadiness(ctx, cancelFunc, task)
	if task.Status == StatusCancelled {
		return nil
	}

	log.Debugf("Running task %q", task.ID)
	task.Status = StatusRunning
	res, err := task.do(ctx, task.Payload)
	saveDoResult(task, &res)

	if err != nil {
		log.Errorf("Task %q failed: %v", task.ID, err)
		task.Status = StatusError
		task.DoError = err
		log.Debugf("Execution cancellation initiated from task %q", task.ID)
		cancelFunc()
		return err
	}

	task.Status = StatusReady
	log.Debugf("Task %q is ready", task.ID)

	for _, notifCh := range task.notifyResult {
		log.Debugf("Sending notification about %v is finished to %v", task.ID, notifCh)
		notifCh <- Result{task.ID, res}
	}
	log.Infof("Task %q execution is finished", task.ID)

	return nil
}
