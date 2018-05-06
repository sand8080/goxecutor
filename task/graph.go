package task

import (
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
)

var ErrNoRootsInGraph = errors.New("no root tasks in graph")

var ErrTaskWaitingParents = errors.New("no root tasks in graph")

var ErrTaskCyclesInGraph = errors.New("tasks cycles in graph")

var ErrTaskUnreached = errors.New("tasks unreached in graph")

type Graph struct {
	sync.RWMutex
	Name           string
	tasks          map[ID]*Task
	roots          map[ID]*Task
	waitingParents map[ID][]*Task
}

func NewGraph(name string) *Graph {
	return &Graph{
		Name:           name,
		tasks:          make(map[ID]*Task),
		roots:          make(map[ID]*Task),
		waitingParents: make(map[ID][]*Task),
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
	from ID
	to   ID
}

// Adds task to the appropriate place in the graph.
func (graph *Graph) Add(task *Task) error {
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
			parentIDs := make([]ID, len(parents))
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
	discover := make(map[ID]taskColor, len(graph.tasks))
	errs := make(map[ID][]tasksCycles, len(graph.roots))

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
		unreached := make([]ID, 0, len(graph.tasks)-len(discover))
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

func (graph *Graph) depthFirstSearch(task *Task, discover map[ID]taskColor, cycles *[]tasksCycles) {
	discover[task.ID] = GREY
	for childID := range task.RequiredFor {
		childColor := discover[childID]
		if childColor == WHITE {
			graph.depthFirstSearch(graph.tasks[childID], discover, cycles)
		} else if childColor == GREY {
			*cycles = append(*cycles, tasksCycles{task.ID, childID})
		}
	}
	discover[task.ID] = BLACK
}
