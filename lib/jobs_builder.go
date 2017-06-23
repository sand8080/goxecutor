package lib

import (
	"errors"
	"fmt"
	"sync"
)

// Builds jobs graph
type JobsBuilder struct {
	sync.RWMutex
	jobs             map[string]*Job
	roots            map[string]*Job
	waitingForParent map[string][]*Job
}

func NewJobsBuilder() *JobsBuilder {
	builder := JobsBuilder{
		jobs:             make(map[string]*Job),
		roots:            make(map[string]*Job),
		waitingForParent: make(map[string][]*Job),
	}
	return &builder
}

// Adds task to the appropriate lib. If checkDuplication is true error will
// be returned on the adding.
func (builder *JobsBuilder) AddJob(job *Job, checkDuplication bool) error {
	builder.Lock()
	defer builder.Unlock()

	if checkDuplication {
		_, ok := builder.jobs[job.task.Name]
		if ok {
			msg := fmt.Sprintf("Task '%s' is already added", job.task.Name)
			return errors.New(msg)
		}
	}

	// Adding job to all jobs map
	builder.jobs[job.task.Name] = job

	// Finding parents
	for req := range job.task.Requires {
		parent, ok := builder.jobs[req]
		if ok {
			// Adding job to already processed parents
			err := parent.addChild(job)
			if err != nil {
				return err
			}
		} else {
			// Registering jobs for adding to parents
			builder.waitingForParent[req] = append(builder.waitingForParent[req],
				job)
		}
	}

	// Finding children
	children, ok := builder.waitingForParent[job.task.Name]
	if ok {
		for _, child := range children {
			err := job.addChild(child)
			if err != nil {
				return err
			}
		}
		delete(builder.waitingForParent, job.task.Name)
	}

	// Adding root jobs
	if job.isRoot() {
		builder.roots[job.task.Name] = job
	}

	return nil
}
