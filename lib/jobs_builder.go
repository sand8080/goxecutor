package lib

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
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

// Job color is used for cycles detection
type jobColor uint8

const WHITE jobColor = 0
const GREY jobColor = 1
const BLACK jobColor = 2

type jobsCycle struct {
	from string
	to   string
}

type jobDiscoverState struct {
	color jobColor
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

func (builder *JobsBuilder) Check() error {
	if err := builder.checkRoots(); err != nil {
		return err
	}
	if err := builder.checkWaitingParents(); err != nil {
		return err
	}
	if err := builder.checkCycles(); err != nil {
		return err
	}
	return nil
}

func (builder *JobsBuilder) checkWaitingParents() error {
	if i := 0; len(builder.waitingForParent) > 0 {
		var buffer bytes.Buffer
		buffer.WriteString("Jobs graph is incomplete. Have waiting for parents jobs: ")
		// Building sorted parents names slice
		parentsNames := make([]string, 0, len(builder.waitingForParent))
		for parentName := range builder.waitingForParent {
			parentsNames = append(parentsNames, parentName)
		}
		sort.Strings(parentsNames)

		for _, parentName := range parentsNames {
			i += 1
			jobs := builder.waitingForParent[parentName]
			jobsNames := make([]string, 0, len(jobs))
			for _, job := range jobs {
				jobsNames = append(jobsNames, job.task.Name)
			}
			msg := fmt.Sprintf("%s is required for: %s", parentName, strings.Join(jobsNames, ", "))
			if len(builder.waitingForParent) > i {
				msg += "; "
			}
			buffer.WriteString(msg)
		}
		return errors.New(buffer.String())
	}
	return nil
}

func (builder *JobsBuilder) checkRoots() error {
	if len(builder.roots) == 0 {
		return errors.New("No root jobs found")
	}
	return nil
}

func (builder *JobsBuilder) checkCycles() error {
	discover := make(map[string]jobColor, len(builder.jobs))
	errs := make(map[string][]jobsCycle, len(builder.roots))

	// Sort root names for fixing root jobs processing order.
	// It is useful for cycles reporting.
	rootNames := make([]string, 0, len(builder.roots))
	for name := range builder.roots {
		rootNames = append(rootNames, name)
	}
	sort.Strings(rootNames)

	for _, root := range builder.roots {
		cycles := make([]jobsCycle, 0)
		builder.depthFirstSearch(root, discover, &cycles)
		if len(cycles) > 0 {
			errs[root.task.Name] = cycles
		}
	}

	if len(errs) > 0 {
		return makeErrorMessage(errs)
	}

	// Checking all jobs are reached
	if len(discover) < len(builder.jobs) {
		unreached := make([]string, 0, len(builder.jobs)-len(discover))
		for jobName := range builder.jobs {
			_, ok := discover[jobName]
			if !ok {
				unreached = append(unreached, jobName)
			}
		}
		sort.Strings(unreached)
		return errors.New(fmt.Sprintf("Following jobs unreached: %s",
			strings.Join(unreached, ", ")))
	}
	return nil
}

func (builder *JobsBuilder) depthFirstSearch(job *Job, discover map[string]jobColor,
	cycles *[]jobsCycle) {
	discover[job.task.Name] = GREY
	for _, child := range job.children {
		child_color := discover[child.task.Name]
		if child_color == WHITE {
			builder.depthFirstSearch(child, discover, cycles)
		} else if child_color == GREY {
			*cycles = append(*cycles, jobsCycle{job.task.Name, child.task.Name})
		}
	}
	discover[job.task.Name] = BLACK
}

func makeErrorMessage(errs map[string][]jobsCycle) error {
	msgs := make([]string, 0, len(errs))
	for rootName, cycles := range errs {
		if len(cycles) >= 0 {
			var b bytes.Buffer
			fmt.Fprintf(&b, "Following cycles are found for the root job %s: ",
				rootName)
			cycle := cycles[0]
			fmt.Fprintf(&b, "from %s to %s", cycle.from, cycle.to)
			for _, cycle := range cycles[1:] {
				fmt.Fprintf(&b, ", from %s to %s", cycle.from, cycle.to)
			}

			msgs = append(msgs, b.String())
		}
	}
	return errors.New(strings.Join(msgs, "\n"))
}
