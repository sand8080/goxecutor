package lib

import (
	"errors"
	"github.com/cznic/mathutil"
	"sort"
	"testing"
)

func checkJobsAreEqual(lh, rh *Job, t *testing.T) {
	if lh.task != rh.task {
		t.Errorf("Jobs %v and %v contains different tasks",
			lh.task.Name, rh.task.Name)
	}
	// Checking parents
	if len(lh.parents) != len(rh.parents) {
		t.Errorf("Jobs %v and %v has different parents count",
			lh.task.Name, rh.task.Name)
	} else {
		for name, lh_parent := range lh.parents {
			rh_parent := rh.parents[name]
			if rh_parent == nil {
				t.Errorf("Job %s has no parent %s. Has parents: %v",
					rh.task.Name, name, rh.parents)
			} else if lh_parent.task != rh_parent.task {
				t.Errorf("Jobs %s, %s has different parent tasks: %v and %v",
					lh.task.Name, rh.task.Name, lh_parent, rh_parent)
			}
		}
	}
	// Checking children
	if len(lh.children) != len(rh.children) {
		t.Errorf("Jobs %v and %v has different children count",
			lh.task.Name, rh.task.Name)
	} else {
		for name, lh_child := range lh.children {
			rh_child := rh.children[name]
			if rh_child == nil {
				t.Errorf("Job %s has no child %s. Has children: %v",
					rh.task.Name, name, rh.children)
			} else if lh_child.task != rh_child.task {
				t.Errorf("Jobs %s, %s has different child tasks: %v and %v",
					lh.task.Name, rh.task.Name, lh_child, rh_child)
			}
		}
	}
}

func TestJobsBuilder_AddTaskInDifferentOrder(t *testing.T) {
	t1 := NewTask("t1", []string{})
	t2 := NewTask("t2", []string{"t1"})
	t3 := NewTask("t3", []string{"t1"})
	t4 := NewTask("t4", []string{"t3", "t2"})
	t5 := NewTask("t5", []string{"t3"})
	tasks := map[string]*Task{
		t1.Name: t1,
		t2.Name: t2,
		t3.Name: t3,
		t4.Name: t4,
		t5.Name: t5,
	}

	exp_j1 := NewJob(t1)
	exp_j2 := NewJob(t2)
	exp_j3 := NewJob(t3)
	exp_j4 := NewJob(t4)
	exp_j5 := NewJob(t5)

	exp_j1.addChild(exp_j2)
	exp_j1.addChild(exp_j3)
	exp_j2.addChild(exp_j4)
	exp_j3.addChild(exp_j4)
	exp_j3.addChild(exp_j5)

	expected_jobs := map[string]*Job{
		t1.Name: exp_j1,
		t2.Name: exp_j2,
		t3.Name: exp_j3,
		t4.Name: exp_j4,
		t5.Name: exp_j5,
	}

	// Checking that in any jobs processing order resulting graph is correct
	taskNames := sort.StringSlice{"t1", "t2", "t3", "t4", "t5"}
	mathutil.PermutationFirst(taskNames)
	hasNext := true
	for hasNext {
		builder := NewJobsBuilder()
		for _, name := range taskNames {
			t := tasks[name]
			builder.AddJob(NewJob(t), false)
		}
		hasNext = mathutil.PermutationNext(taskNames)

		for _, name := range taskNames {
			expected_job := expected_jobs[name]
			actual_job := builder.jobs[name]
			checkJobsAreEqual(expected_job, actual_job, t)
		}
	}
}

func TestJobsBuilder_checkWaitingParents(t *testing.T) {
	cases := []struct {
		tasks    []*Task
		hasError bool
	}{
		{
			tasks:    []*Task{},
			hasError: false,
		},
		{
			tasks: []*Task{
				NewTask("root", []string{}),
			},
			hasError: false,
		},
		{
			tasks: []*Task{
				NewTask("root", []string{}),
				NewTask("child1", []string{"root"}),
				NewTask("child2", []string{"root"}),
			},
			hasError: false,
		},
		{
			tasks: []*Task{
				NewTask("child1", []string{"root"}),
				NewTask("child2", []string{"root"}),
				NewTask("root", []string{}),
			},
			hasError: false,
		},
		{
			tasks: []*Task{
				NewTask("child1", []string{"root"}),
				NewTask("child2", []string{"root"}),
			},
			hasError: true,
		},
	}

	for _, c := range cases {
		builder := NewJobsBuilder()
		for _, t := range c.tasks {
			builder.AddJob(NewJob(t), false)
		}
		if (builder.checkWaitingParents() != nil) != c.hasError {
			t.Errorf("Tasks set has expected error state: %b. Tasks: %v",
				c.hasError, c.tasks)
		}

	}
}

func TestJobsBuilder_checkRoots(t *testing.T) {
	cases := []struct {
		jobs []*Job
		err  string
	}{
		{
			jobs: []*Job{},
			err:  "No root jobs found",
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{"t1"})),
			},
			err: "No root jobs found",
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{"t2"})),
				NewJob(NewTask("t2", []string{"t1"})),
			},
			err: "No root jobs found",
		},
	}

	for _, c := range cases {
		builder := NewJobsBuilder()
		for _, job := range c.jobs {
			builder.AddJob(job, true)
		}
		if act := builder.checkRoots(); act == nil || c.err != act.Error() {
			t.Errorf("Expected error: %v, got: %v", c.err, act)
		}

	}
}

func TestJobsBuilder_checkCycles(t *testing.T) {
	cases := []struct {
		jobs []*Job
		err  string
	}{
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{})),
				NewJob(NewTask("t2", []string{"t1", "t3"})),
				NewJob(NewTask("t3", []string{"t2"})),
			},
			err: "Following cycles are found for the root job t1: from t3 to t2",
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{})),
				NewJob(NewTask("t2", []string{"t1", "t2"})),
			},
			err: "Following cycles are found for the root job t1: from t2 to t2",
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{})),
				NewJob(NewTask("t2", []string{"t1", "t4"})),
				NewJob(NewTask("t3", []string{"t2"})),
				NewJob(NewTask("t4", []string{"t3"})),
			},
			err: "Following cycles are found for the root job t1: from t4 to t2",
		},
	}
	for _, c := range cases {
		builder := NewJobsBuilder()
		for _, job := range c.jobs {
			err := builder.AddJob(job, true)
			if err != nil {
				t.Error(err)
			}
		}
		if act := builder.checkCycles(); act == nil || c.err != act.Error() {
			t.Errorf("Expected error: %v, got: %v", c.err, act)
		}
	}
}

func TestJobsBuilder_Check(t *testing.T) {
	cases := []struct {
		jobs []*Job
		err  error
	}{
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{})),
				NewJob(NewTask("t2", []string{"t1"})),
				NewJob(NewTask("t3", []string{"t2"})),
			},
			err: nil,
		},
		{
			jobs: []*Job{
				NewJob(NewTask("root1", []string{})),
				NewJob(NewTask("root2", []string{})),
				NewJob(NewTask("child20", []string{"root2"})),
			},
			err: nil,
		},
		{
			jobs: []*Job{},
			err:  errors.New("No root jobs found"),
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t2", []string{"t1"})),
			},
			err: errors.New("No root jobs found"),
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{})),
				NewJob(NewTask("t3", []string{"t2", "t4"})),
				NewJob(NewTask("t5", []string{"t2"})),
			},
			err: errors.New("Jobs graph is incomplete. " +
				"Have waiting for parents jobs: t2 is required for: t3, t5; " +
				"t4 is required for: t3"),
		},
		{
			jobs: []*Job{
				NewJob(NewTask("t1", []string{})),
				NewJob(NewTask("t2", []string{"t3", "t1"})),
				NewJob(NewTask("t3", []string{"t2"})),
				NewJob(NewTask("t4", []string{"t1", "t4"})),
			},
			err: errors.New("Following cycles are found for the root job t1: from t3 to t2, " +
				"from t4 to t4"),
		},
	}
	for _, c := range cases {
		builder := NewJobsBuilder()
		for _, job := range c.jobs {
			err := builder.AddJob(job, true)
			if err != nil {
				t.Error(err)
			}
		}
		act := builder.Check()
		if c.err == nil {
			if act != nil {
				t.Errorf("Expected error: %v, got: %v", c.err, act)
			}
		} else {
			if act == nil || c.err.Error() != act.Error() {
				t.Errorf("Expected error: %v, got: %v", c.err, act)
			}
		}
	}
}
