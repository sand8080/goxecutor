package task

import (
	"errors"
	"fmt"
)

type Task struct {
	Name     string
	Priority int
	requires map[string]bool
	children map[string]*Task
}

func NewTask(name string, requires []string) *Task {
	t := Task{
		Name:     name,
		requires: make(map[string]bool, len(requires)),
		children: make(map[string]*Task, len(requires)),
	}
	for _, req := range requires {
		t.requires[req] = true
	}
	return &t
}

func (t *Task) IsRoot() bool {
	return len(t.requires) == 0
}

func (t *Task) AddChild(child *Task) error {
	_, ok := child.requires[t.Name]
	if ok == false {
		return errors.New(fmt.Sprintf("Task '%s' can't be parent for '%s'. "+
			"It doesn't require '%s'", t.Name, child.Name, t.Name))
	}
	t.children[child.Name] = child
	return nil
}

type TasksByPriority []*Task

func (tasks TasksByPriority) Len() int { return len(tasks) }

func (tasks TasksByPriority) Swap(i, j int) {
	tasks[i], tasks[j] = tasks[j], tasks[i]
}

func (tasks TasksByPriority) Less(i, j int) bool {
	return tasks[i].Priority < tasks[j].Priority
}
