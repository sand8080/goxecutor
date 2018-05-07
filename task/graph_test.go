package task_test

import (
	"sort"
	"testing"

	"github.com/cznic/mathutil"
	. "github.com/sand8080/goxecutor/task"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGraph_AddTaskInDifferentOrder(t *testing.T) {
	tasksFunc := func() map[ID]*Task {
		t1 := NewTask("t1", nil, nil, nil, nil)
		t2 := NewTask("t2", []ID{"t1"}, nil, nil, nil)
		t3 := NewTask("t3", []ID{"t1"}, nil, nil, nil)
		t4 := NewTask("t4", []ID{"t3", "t2"}, nil, nil, nil)
		t5 := NewTask("t5", []ID{"t3"}, nil, nil, nil)
		tasks := map[ID]*Task{
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
			assert.NoError(t, graph.Add(tasks[ID(IDString)]))
		}
		hasNext = mathutil.PermutationNext(taskIDStrings)
		assert.NoError(t, graph.Check())
	}
}

func TestGraph_checkWaitingParents(t *testing.T) {
	cases := []struct {
		tasks  []*Task
		expErr error
	}{
		{
			tasks:  []*Task{},
			expErr: nil,
		},
		{
			tasks: []*Task{
				NewTask("root", nil, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*Task{
				NewTask("root", nil, nil, nil, nil),
				NewTask("child1", []ID{"root"}, nil, nil, nil),
				NewTask("child2", []ID{"root"}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*Task{
				NewTask("child1", []ID{"root"}, nil, nil, nil),
				NewTask("child2", []ID{"root"}, nil, nil, nil),
				NewTask("root", []ID{}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*Task{
				NewTask("child1", []ID{"root"}, nil, nil, nil),
				NewTask("child2", []ID{"root"}, nil, nil, nil),
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
		tasks  []*Task
		expErr error
	}{
		{
			tasks:  []*Task{},
			expErr: ErrNoRootsInGraph,
		},
		{
			tasks: []*Task{
				NewTask("t1", []ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
		{
			tasks: []*Task{
				NewTask("t1", []ID{"t2"}, nil, nil, nil),
				NewTask("t2", []ID{"t1"}, nil, nil, nil),
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
		tasks  []*Task
		expErr error
	}{
		{
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t1", "t3"}, nil, nil, nil),
				NewTask("t3", []ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// Cycle from t2 to t2
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t1", "t2"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// Cycle from t4 to t2
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t1", "t4"}, nil, nil, nil),
				NewTask("t3", []ID{"t2"}, nil, nil, nil),
				NewTask("t4", []ID{"t3"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// Unreached tasks: t2, t3, t4, t5
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t4"}, nil, nil, nil),
				NewTask("t3", []ID{"t2"}, nil, nil, nil),
				NewTask("t4", []ID{"t3"}, nil, nil, nil),
				NewTask("t5", []ID{"t5"}, nil, nil, nil),
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
		tasks  []*Task
		expErr error
	}{
		{
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t1"}, nil, nil, nil),
				NewTask("t3", []ID{"t2"}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks: []*Task{
				NewTask("root1", nil, nil, nil, nil),
				NewTask("root2", nil, nil, nil, nil),
				NewTask("child20", []ID{"root2"}, nil, nil, nil),
			},
			expErr: nil,
		},
		{
			tasks:  []*Task{},
			expErr: ErrNoRootsInGraph,
		},
		{
			tasks: []*Task{
				NewTask("t2", []ID{"t1"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
		{
			// t2 waiting t3, t5
			// t4 waiting t3
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t3", []ID{"t2", "t4"}, nil, nil, nil),
				NewTask("t5", []ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrNoRootsInGraph,
		},
		{
			// Cycle for t1: from t3 to t2 and from t4 to t4
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t3", "t1"}, nil, nil, nil),
				NewTask("t3", []ID{"t2"}, nil, nil, nil),
				NewTask("t4", []ID{"t1", "t4"}, nil, nil, nil),
			},
			expErr: ErrTaskCyclesInGraph,
		},
		{
			// t2 unreached
			tasks: []*Task{
				NewTask("t1", nil, nil, nil, nil),
				NewTask("t2", []ID{"t2"}, nil, nil, nil),
			},
			expErr: ErrTaskUnreached,
		},
	}
	for _, c := range cases {
		graph := NewGraph("")
		for _, task := range c.tasks {
			assert.NoError(t, graph.Add(task))
		}
		actErr := graph.Check()
		assert.Equal(t, c.expErr, actErr, "Graph checking expected error: %v, got: %v",
			c.expErr, actErr)
	}
}
