package task

import (
	"errors"
	"fmt"
	"sync"
)

// Builds jobs from tasks. Job is graph of tasks with
// defined root task
type JobsBuilder struct {
	sync.RWMutex
	tasks            map[string]*Task
	roots            map[string]*Task
	waitingForParent map[string][]*Task
}

func NewJobsBuilder() *JobsBuilder {
	builder := JobsBuilder{
		tasks:            make(map[string]*Task),
		roots:            make(map[string]*Task),
		waitingForParent: make(map[string][]*Task),
	}
	return &builder
}

// Adds task to the appropriate job. If checkDuplication is true error will
// be returned on the adding.
func (builder *JobsBuilder) AddTask(task *Task, checkDuplication bool) error {
	builder.Lock()
	defer builder.Unlock()

	if checkDuplication {
		_, ok := builder.tasks[task.Name]
		if ok {
			msg := fmt.Sprintf("Task '%s' is already added", task.Name)
			return errors.New(msg)
		}
	}

	// Adding task to all tasks map
	builder.tasks[task.Name] = task

	// Finding parents
	for req := range task.requires {
		parent, ok := builder.tasks[req]
		if ok {
			// Adding task to already processed parents
			parent.AddChild(task)
			//builder.tasks[req] = parent
		} else {
			// Registering tasks for adding to parents
			builder.waitingForParent[req] = append(builder.waitingForParent[req],
				task)
		}
	}

	// Finding children
	children, ok := builder.waitingForParent[task.Name]
	if ok {
		for _, child := range children {
			err := task.AddChild(child)
			if err != nil {
				return err
			}
		}
		delete(builder.waitingForParent, task.Name)
	}

	return nil
}
