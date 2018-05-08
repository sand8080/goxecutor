package graph

import (
	"context"
	"errors"
	"sort"
	"sync"
	"testing"

	"github.com/cznic/mathutil"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/sand8080/goxecutor/task"
)

type MapStorage struct {
	sync.Mutex
	tasks map[uuid.UUID]*task.Task
}

func newMapStorage() *MapStorage {
	return &MapStorage{tasks: make(map[uuid.UUID]*task.Task)}
}

func (s *MapStorage) Save(t *task.Task) error {
	s.Lock()
	defer s.Unlock()

	if t.UUID == uuid.Nil {
		t.UUID = uuid.NewV4()
	}
	s.tasks[t.UUID] = t
	return nil
}

func (s *MapStorage) Load(uuid uuid.UUID) (*task.Task, error) {
	s.Lock()
	defer s.Unlock()

	t, ok := s.tasks[uuid]
	if !ok {
		return nil, errors.New("task not found in storage")
	}

	return t, nil
}

func doNothing(ctx context.Context, payload interface{}) (interface{}, error) {
	return nil, nil
}

func TestGraph_AddTaskInDifferentOrder(t *testing.T) {
	tasksFunc := func() map[task.ID]*task.Task {
		t1 := task.NewTask("t1", nil, nil, nil, nil)
		t2 := task.NewTask("t2", []task.ID{"t1"}, nil, nil, nil)
		t3 := task.NewTask("t3", []task.ID{"t1"}, nil, nil, nil)
		t4 := task.NewTask("t4", []task.ID{"t3", "t2"}, nil, nil, nil)
		t5 := task.NewTask("t5", []task.ID{"t3"}, nil, nil, nil)
		tasks := map[task.ID]*task.Task{
			t1.ID: t1,
			t2.ID: t2,
			t3.ID: t3,
			t4.ID: t4,
			t5.ID: t5,
		}
		return tasks
	}

	// Checking that in any tasks processing order resulting graph is correct
	taskIDStrings := sort.StringSlice{"t1", "t2", "t3", "t4", "t5"}
	mathutil.PermutationFirst(taskIDStrings)
	hasNext := true
	for hasNext {
		graph := NewGraph("")
		tasks := tasksFunc()
		log.Debugf("Tasks IDs: %v", taskIDStrings)
		for _, IDString := range taskIDStrings {
			assert.NoError(t, graph.Add(tasks[task.ID(IDString)]))
		}
		hasNext = mathutil.PermutationNext(taskIDStrings)
		assert.NoError(t, graph.Check())
	}
}

func TestGraph_checkWaitingParents(t *testing.T) {
	cases := []struct {
		tasks  []*task.Task
		expErr error
	}{
		{
			tasks:  []*task.Task{},
			expErr: nil,
		},
		{
			tasks: []*task.Task{
				task.NewTask("root", nil, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*task.Task{
				task.NewTask("root", nil, nil, nil, nil),
				task.NewTask("child1", []task.ID{"root"}, nil, nil, nil),
				task.NewTask("child2", []task.ID{"root"}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*task.Task{
				task.NewTask("child1", []task.ID{"root"}, nil, nil, nil),
				task.NewTask("child2", []task.ID{"root"}, nil, nil, nil),
				task.NewTask("root", []task.ID{}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*task.Task{
				task.NewTask("child1", []task.ID{"root"}, nil, nil, nil),
				task.NewTask("child2", []task.ID{"root"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
	}

	for _, c := range cases {
		graph := NewGraph("")
		for _, task := range c.tasks {
			assert.NoError(t, graph.Add(task))
		}
		actErr := graph.CheckWaitingParents()
		assert.Equal(t, c.expErr, actErr, "Checking of waiting parents expected error: %v, got: %v",
			c.expErr, actErr)

	}
}

func TestJobsBuilder_CheckRoots(t *testing.T) {
	cases := []struct {
		tasks  []*task.Task
		expErr error
	}{
		{
			tasks:  []*task.Task{},
			expErr: ErrNoRootsInGraph,
		},
		{
			tasks: []*task.Task{
				task.NewTask("t1", []task.ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
		{
			tasks: []*task.Task{
				task.NewTask("t1", []task.ID{"t2"}, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t1"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
	}

	for _, c := range cases {
		graph := NewGraph("")
		for _, task := range c.tasks {
			assert.NoError(t, graph.Add(task))
		}
		if actErr := graph.CheckRoots(); actErr == nil || c.expErr != actErr {
			t.Errorf("Expected error: %v, got: %v", c.expErr, actErr)
		}

	}
}

func TestGraph_CheckCycles(t *testing.T) {
	cases := []struct {
		tasks  []*task.Task
		expErr error
	}{
		{
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t1", "t3"}, nil, nil, nil),
				task.NewTask("t3", []task.ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// Cycle from t2 to t2
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t1", "t2"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// Cycle from t4 to t2
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t1", "t4"}, nil, nil, nil),
				task.NewTask("t3", []task.ID{"t2"}, nil, nil, nil),
				task.NewTask("t4", []task.ID{"t3"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// Unreached tasks: t2, t3, t4, t5
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t4"}, nil, nil, nil),
				task.NewTask("t3", []task.ID{"t2"}, nil, nil, nil),
				task.NewTask("t4", []task.ID{"t3"}, nil, nil, nil),
				task.NewTask("t5", []task.ID{"t5"}, nil, nil, nil),
			},
			expErr: ErrTaskUnreached,
		},
	}
	for _, c := range cases {
		graph := NewGraph("")
		for _, task := range c.tasks {
			assert.NoError(t, graph.Add(task))
		}
		actErr := graph.CheckCycles()
		assert.Equal(t, c.expErr, actErr, "Checking cycles expected error: %v, got: %v",
			c.expErr, actErr)
	}
}

func TestGraph_Check(t *testing.T) {
	cases := []struct {
		tasks  []*task.Task
		expErr error
	}{
		{
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t1"}, nil, nil, nil),
				task.NewTask("t3", []task.ID{"t2"}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*task.Task{
				task.NewTask("root1", nil, nil, nil, nil),
				task.NewTask("root2", nil, nil, nil, nil),
				task.NewTask("child20", []task.ID{"root2"}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks:  []*task.Task{},
			expErr: ErrNoRootsInGraph,
		},
		{
			tasks: []*task.Task{
				task.NewTask("t2", []task.ID{"t1"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
		{
			// t2 waiting t3, t5
			// t4 waiting t3
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t3", []task.ID{"t2", "t4"}, nil, nil, nil),
				task.NewTask("t5", []task.ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
		{
			// Cycle for t1: from t3 to t2 and from t4 to t4
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t3", "t1"}, nil, nil, nil),
				task.NewTask("t3", []task.ID{"t2"}, nil, nil, nil),
				task.NewTask("t4", []task.ID{"t1", "t4"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// t2 unreached
			tasks: []*task.Task{
				task.NewTask("t1", nil, nil, nil, nil),
				task.NewTask("t2", []task.ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrTaskUnreached,
		},
	}
	for _, c := range cases {
		graph := NewGraph("")
		for _, tsk := range c.tasks {
			assert.NoError(t, graph.Add(tsk))
		}
		actErr := graph.Check()
		assert.Equal(t, c.expErr, actErr, "Graph checking expected error: %v, got: %v",
			c.expErr, actErr)
	}
}

func TestGraph_Exec(t *testing.T) {
	graph := NewGraph("")
	parent := task.NewTask("P", nil, nil, doNothing, doNothing)
	child := task.NewTask("CH", []task.ID{"P"}, nil, doNothing, doNothing)
	graph.Add(parent)
	graph.Add(child)

	storage := newMapStorage()
	status, err := graph.Exec(PolicyIgnoreError, storage)

	assert.NoError(t, err)
	assert.Equal(t, StatusSuccess, status)
}
