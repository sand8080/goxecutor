package lib

import (
	"reflect"
	"strings"
	"testing"
)

func TestJob_isRoot(t *testing.T) {
	cases := []struct {
		job      *Job
		expected bool
	}{
		{job: NewJob(NewTask("A", []string{})), expected: true},
		{job: NewJob(NewTask("A", []string{"B"})), expected: false},
	}
	for _, c := range cases {
		actual := c.job.isRoot()
		if actual != c.expected {
			t.Errorf("Expected isRoot value '%v' for task %v",
				c.expected, c.job.task)
		}
	}
}

func TestJob_addChild(t *testing.T) {
	parent := NewJob(NewTask("job", []string{}))
	child := NewJob(NewTask("child", []string{"job"}))

	err := parent.addChild(child)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parent.children, map[string]*Job{"child": child}) {
		t.Errorf("Child %v wasn't added correctly to %v", child, parent)
	}
}

func TestJob_addChildSelfLoop(t *testing.T) {
	job := NewJob(NewTask("job", []string{"job"}))

	err := job.addChild(job)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(job.children, map[string]*Job{"job": job}) {
		t.Errorf("Self looped job creation failed")
	}
	if !reflect.DeepEqual(job.parents, map[string]*Job{"job": job}) {
		t.Errorf("Self looped job creation failed")
	}
}

func TestJob_addChildCheckParent(t *testing.T) {
	parent := NewJob(NewTask("parent", []string{}))
	child := NewJob(NewTask("child", []string{"parent"}))

	err := parent.addChild(child)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parent.children, map[string]*Job{"child": child}) {
		t.Errorf("Child %v wasn't added correctly to %v", child, parent)
	}

	if len(parent.parents) > 0 {
		t.Error("Root job shouldn't have parents")
	}
	_, hasParent := child.parents[parent.task.Name]
	if !hasParent {
		t.Error("Parent doesn't set to child")
	}
}

func TestJob_addChildDup(t *testing.T) {
	parent := NewJob(NewTask("job", []string{}))
	child := NewJob(NewTask("child", []string{"job"}))
	child_dup := NewJob(NewTask("child", []string{"job"}))

	err := parent.addChild(child)
	if err != nil {
		t.Error(err)
	}
	err = parent.addChild(child_dup)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parent.children, map[string]*Job{"child": child}) {
		t.Errorf("Child %v wasn't added correctly to %v", child, child_dup)
	}
}

func TestJob_addChildError(t *testing.T) {
	one := NewJob(NewTask("one", []string{}))
	two := NewJob(NewTask("two", []string{}))

	err := one.addChild(two)
	if !strings.Contains(err.Error(), "can't be child") {
		t.Error("It is possible to add task without requirement as child")
	}
}
