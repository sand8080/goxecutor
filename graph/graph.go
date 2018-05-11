package graph

import (
	"errors"
	"sync"

	"context"
	log "github.com/sirupsen/logrus"

	"github.com/sand8080/goxecutor/task"
)

var ErrNoRootsInGraph = errors.New("no root tasks in graph")

var ErrTaskWaitingParents = errors.New("no root tasks in graph")

var ErrTaskCyclesInGraph = errors.New("tasks cycles in graph")

var ErrTaskUnreached = errors.New("tasks unreached in graph")

var ErrPolicyNotHandled = errors.New("execution policy not handled")

type Graph struct {
	sync.RWMutex
	Name           string
	tasks          map[task.ID]*task.Task
	roots          map[task.ID]*task.Task
	waitingParents map[task.ID][]*task.Task
}

type ExecutionStatus string

const (
	StatusSuccess   ExecutionStatus = "SUCCESS"
	StatusError     ExecutionStatus = "ERROR"
	StatusCancelled ExecutionStatus = "CANCELLED"
)

type ExecutionPolicy string

const (
	PolicyRevertOnError ExecutionPolicy = "REVERT_ON_ERROR"
	PolicyIgnoreError   ExecutionPolicy = "IGNORE_ERROR"
)

func NewGraph(name string) *Graph {
	return &Graph{
		Name:           name,
		tasks:          make(map[task.ID]*task.Task),
		roots:          make(map[task.ID]*task.Task),
		waitingParents: make(map[task.ID][]*task.Task),
	}
}

// Task color for cycles detection
type taskColor uint8

const (
	WHITE taskColor = iota
	GREY
	BLACK
)

type tasksCycles struct {
	from task.ID
	to   task.ID
}

// Adds task to the appropriate place in the graph.
func (graph *Graph) Add(task *task.Task) error {
	graph.Lock()
	defer graph.Unlock()

	// Adding task to all tasks map
	graph.tasks[task.ID] = task

	// Finding parents
	for reqID := range task.Requires {
		if parent, ok := graph.tasks[reqID]; ok {
			// Adding task parents already in the graph
			if err := parent.AddChild(task); err != nil {
				return err
			}
		} else {
			// Registering task for adding to parents
			graph.waitingParents[reqID] = append(graph.waitingParents[reqID], task)
		}
	}

	// Finding children
	if children, ok := graph.waitingParents[task.ID]; ok {
		for _, child := range children {
			if err := task.AddChild(child); err != nil {
				return err
			}
		}
		delete(graph.waitingParents, task.ID)
	}

	// Adding root tasks
	if len(task.Requires) == 0 {
		graph.roots[task.ID] = task
	}

	return nil
}

func (graph *Graph) Check() error {
	if err := graph.CheckRoots(); err != nil {
		return err
	}
	if err := graph.CheckWaitingParents(); err != nil {
		return err
	}
	if err := graph.CheckCycles(); err != nil {
		return err
	}
	return nil
}

func (graph *Graph) CheckWaitingParents() error {
	if len(graph.waitingParents) > 0 {
		for childID, parents := range graph.waitingParents {
			parentIDs := make([]task.ID, len(parents))
			for idx, parent := range parents {
				parentIDs[idx] = parent.ID
			}
			log.Errorf("Task %v waiting for parents: %v", childID, parentIDs)
		}
		return ErrTaskWaitingParents
	}
	return nil
}

func (graph *Graph) CheckRoots() error {
	if len(graph.roots) == 0 {
		return ErrNoRootsInGraph
	}
	return nil
}

func (graph *Graph) CheckCycles() error {
	discover := make(map[task.ID]taskColor, len(graph.tasks))
	errs := make(map[task.ID][]tasksCycles, len(graph.roots))

	for _, root := range graph.roots {
		cycles := make([]tasksCycles, 0)
		graph.depthFirstSearch(root, discover, &cycles)
		if len(cycles) > 0 {
			errs[root.ID] = cycles
			log.Errorf("Cycles for root %v found: %v", root.ID, cycles)
		}
	}

	if len(errs) > 0 {
		return ErrTaskCyclesInGraph
	}

	// Checking all tasks are reached
	if len(discover) < len(graph.tasks) {
		unreached := make([]task.ID, 0, len(graph.tasks)-len(discover))
		for taskID := range graph.tasks {
			_, ok := discover[taskID]
			if !ok {
				unreached = append(unreached, taskID)
			}
		}
		log.Errorf("Following tasks unreached: %v", unreached)
		return ErrTaskUnreached
	}

	return nil
}

func (graph *Graph) depthFirstSearch(t *task.Task, discover map[task.ID]taskColor, cycles *[]tasksCycles) {
	discover[t.ID] = GREY
	for childID := range t.RequiredFor {
		childColor := discover[childID]
		if childColor == WHITE {
			graph.depthFirstSearch(graph.tasks[childID], discover, cycles)
		} else if childColor == GREY {
			*cycles = append(*cycles, tasksCycles{t.ID, childID})
		}
	}
	discover[t.ID] = BLACK
}

func (graph *Graph) tasksStatuses() map[task.Status][]task.ID {
	result := make(map[task.Status][]task.ID)
	for _, task := range graph.tasks {
		result[task.Status] = append(result[task.Status], task.ID)
	}
	return result
}

func (graph *Graph) Exec(policy ExecutionPolicy, storage task.Storage) (ExecutionStatus, error) {
	var wg sync.WaitGroup
	ctx, cancelFunc := context.WithCancel(context.Background())

	doTask := func(t *task.Task) {
		task.Do(ctx, cancelFunc, t, storage)
		wg.Done()
	}

	for _, t := range graph.tasks {
		wg.Add(1)
		go doTask(t)
	}

	wg.Wait()

	// TODO(sand8080): handle execution policies
	tasksStatuses := graph.tasksStatuses()
	switch policy {
	case PolicyIgnoreError:
		var s ExecutionStatus
		if len(tasksStatuses[task.StatusError]) > 0 {
			s = StatusError
		} else {
			s = StatusSuccess
		}
		return s, nil
	default:
		log.Errorf("Execution policy %v doesn't handled", policy)
		return "", ErrPolicyNotHandled
	}
}
