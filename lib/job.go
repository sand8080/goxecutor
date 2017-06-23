package lib

import (
	"errors"
	"fmt"
	"sync"
)

// Task is execution primitive
type Task struct {
	Name     string
	Priority int
	Requires map[string]bool
}

func NewTask(name string, requires []string) *Task {
	t := Task{
		Name:     name,
		Requires: make(map[string]bool, len(requires)),
	}
	for _, req := range requires {
		t.Requires[req] = true
	}
	return &t
}

// Job is graph node for task. Jobs are bound between
// themselves as parent and children
type Job struct {
	sync.RWMutex
	task     *Task
	parents  map[string]*Job
	children map[string]*Job
}

func NewJob(task *Task) *Job {
	job := Job{
		task:     task,
		parents:  make(map[string]*Job),
		children: make(map[string]*Job, len(task.Requires)),
	}
	return &job
}

func (job *Job) isRoot() bool {
	return len(job.task.Requires) == 0
}

func (job *Job) addChild(child *Job) error {
	_, ok := child.task.Requires[job.task.Name]
	if ok == false {
		return errors.New(fmt.Sprintf(
			"Job '%s' can't be child for '%s'. It doesn't require '%s'",
			job.task.Name, child.task.Name, job.task.Name))
	}

	job.Lock()
	defer job.Unlock()
	child.Lock()
	defer child.Unlock()

	job.children[child.task.Name] = child
	child.parents[job.task.Name] = job

	return nil
}
