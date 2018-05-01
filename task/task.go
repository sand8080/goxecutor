package task

import (
	"context"
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
)

type ID string

type Status string

const (
	StatusNew       = Status("NEW")
	StatusWaiting   = Status("WAITING")
	StatusRunning   = Status("RUNNING")
	StatusReady     = Status("READY")
	StatusCancelled = Status("CANCELLED")
	StatusError     = Status("ERROR")
)

var ErrNotChild = errors.New("adding non required child")

var ErrChildAlreadyAdded = errors.New("child already added")

// DoFunc is task payload processing function.
// If DoFunc returns not nil value and there are no error the value  will be saved in context as value
// with task.ID as key.
type DoFunc func(ctx context.Context, payload interface{}) (interface{}, error)

type taskResult struct {
	id     ID
	result interface{}
}

type Task struct {
	sync.Mutex
	ID            ID
	Status        Status
	Requires      map[ID]bool
	Payload       interface{}
	do            DoFunc
	RequiredFor   map[ID]bool
	waitingResult chan taskResult
	notifyResult  []chan<- taskResult
}

func NewTask(id ID, requires []ID, payload interface{}, doFunc DoFunc) *Task {
	reqSet := make(map[ID]bool, len(requires))
	for _, req := range requires {
		reqSet[req] = true
	}
	var waitingID chan taskResult
	if len(requires) > 0 {
		waitingID = make(chan taskResult, len(requires))
	}
	return &Task{
		ID:            id,
		Status:        StatusNew,
		Requires:      reqSet,
		RequiredFor:   make(map[ID]bool),
		Payload:       payload,
		do:            doFunc,
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

func Exec(ctx context.Context, cancelFunc context.CancelFunc, task *Task) error {
	// Task lock prevents events mess up in case of multiple Exec calls with the same task object.
	task.Lock()
	defer task.Unlock()

	log.Debugf("Execution of task %q initiated", task.ID)

	if task.waitingResult != nil {
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
				return nil
			}
		}
	}

	log.Debugf("Running task %q", task.ID)
	task.Status = StatusRunning

	res, err := task.do(ctx, task.Payload)
	if err != nil {
		log.Errorf("Task %q failed: %v", task.ID, err)
		task.Status = StatusError
		log.Debugf("Execution cancellation initiated from task %q", task.ID)
		cancelFunc()
		return err
	}

	task.Status = StatusReady
	log.Debugf("Task %q is ready", task.ID)

	for _, notifCh := range task.notifyResult {
		log.Debugf("Sending notification about %v is finished to %v", task.ID, notifCh)
		notifCh <- taskResult{task.ID, res}
	}
	log.Infof("Task %q execution is finished", task.ID)

	return nil
}
