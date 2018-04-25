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

type DoFunc func(payload interface{}) error

type Task struct {
	sync.Mutex
	ID          ID
	Status      Status
	Requires    map[ID]bool
	Payload     interface{}
	do          DoFunc
	RequiredFor map[ID]bool
	waitingID   chan ID
	notifyIDs   []chan<- ID
}

func NewTask(id ID, requires []ID, payload interface{}, execFunc DoFunc) *Task {
	reqSet := make(map[ID]bool, len(requires))
	for _, req := range requires {
		reqSet[req] = true
	}
	var waitingID chan ID
	if len(requires) > 0 {
		waitingID = make(chan ID, len(requires))
	}
	return &Task{
		ID:          id,
		Status:      StatusNew,
		Requires:    reqSet,
		RequiredFor: make(map[ID]bool),
		Payload:     payload,
		do:          execFunc,
		waitingID:   waitingID,
	}
}

// AddChild adds child task.
func (t *Task) AddChild(child Task) error {
	t.Lock()
	defer t.Unlock()

	log.Debugf("Adding child %q to %q", child.ID, t.ID)
	if !child.Requires[t.ID] {
		return ErrNotChild
	}

	if added, ok := t.RequiredFor[child.ID]; ok && added {
		return ErrChildAlreadyAdded
	}

	t.notifyIDs = append(t.notifyIDs, child.waitingID)
	t.RequiredFor[child.ID] = true
	return nil
}

func Exec(ctx context.Context, cancelFunc context.CancelFunc, task *Task) error {
	// Task lock prevents events mess up in case of multiple Exec calls with the same task object.
	task.Lock()
	defer task.Unlock()

	log.Debugf("Execution of task %q initiated", task.ID)

	if task.waitingID != nil {
		task.Status = StatusWaiting
		received := make(map[ID]bool, len(task.Requires))
	loop:
		for {
			select {
			case id := <-task.waitingID:
				log.Debugf("%v is notified: task %q is finished.", task.ID, id)
				received[id] = true

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

	log.Debugf("Running task %q", task.ID)
	task.Status = StatusRunning
	if err := task.do(task.Payload); err != nil {
		log.Errorf("Task %q failed: %v", task.ID, err)
		task.Status = StatusError
		log.Debugf("Execution cancellation initiated from task %q", task.ID)
		cancelFunc()
		return err
	}

	task.Status = StatusReady
	log.Debugf("Task %q is ready", task.ID)

	for _, notifCh := range task.notifyIDs {
		log.Debugf("Sending notification about %v is finished to %v", task.ID, notifCh)
		notifCh <- task.ID
	}
	log.Infof("Task %q execution is finished", task.ID)

	return nil
}
