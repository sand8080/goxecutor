package task

import (
	"reflect"
	"strings"
	"testing"
)

func TestTask_IsRoot(t *testing.T) {
	cases := []struct {
		task     Task
		expected bool
	}{
		{task: Task{Name: "A"}, expected: true},
		{task: Task{Name: "A", requires: map[string]bool{"B": false}},
			expected: false},
	}
	for _, c := range cases {
		actual := c.task.IsRoot()
		if actual != c.expected {
			t.Errorf("Expected IsRoot value '%v' for task %v",
				c.expected, c.task)
		}
	}
}

func TestTask_AddChild(t *testing.T) {
	parent := NewTask("parent", []string{})
	child := NewTask("child", []string{"parent"})

	err := parent.AddChild(child)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parent.children, map[string]*Task{"child": child}) {
		t.Errorf("Child %v wasn't added correctly to %v", child, parent)
	}
}

func TestTask_AddChildDup(t *testing.T) {
	parent := NewTask("parent", []string{})
	child := NewTask("child", []string{"parent"})
	child_dup := NewTask("child", []string{"parent"})

	err := parent.AddChild(child)
	if err != nil {
		t.Error(err)
	}
	err = parent.AddChild(child_dup)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(parent.children, map[string]*Task{"child": child}) {
		t.Errorf("Child %v wasn't added correctly to %v", child, child_dup)
	}
}

func TestTask_AddChildError(t *testing.T) {
	one := NewTask("one", []string{})
	two := NewTask("two", []string{})

	err := one.AddChild(two)
	if !strings.Contains(err.Error(), "can't be parent") {
		t.Error("It is possible to add task without requirement as child")
	}
}
