package lib

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
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

func (job *Job) String() string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Job %q\n", job.task.Name)

	parents := make([]string, 0, len(job.parents))
	for p := range job.parents {
		parents = append(parents, p)
	}
	fmt.Fprintf(&b, "\tparents: %s\n", strings.Join(parents, ","))

	children := make([]string, 0, len(job.children))
	for c := range job.children {
		children = append(children, c)
	}
	fmt.Fprintf(&b, "\tchildren: %s", strings.Join(children, ","))

	return b.String()
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
	// In case of self loop we shouldn't lock job again
	if job != child {
		child.Lock()
		defer child.Unlock()
	}

	job.children[child.task.Name] = child
	child.parents[job.task.Name] = job

	return nil
}
