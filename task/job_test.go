package task

import "testing"

func TestJobsBuilder_AddTask(t *testing.T) {
	t1 := NewTask("t1", []string{})
	t2 := NewTask("t2", []string{"t1"})
	t3 := NewTask("t3", []string{"t1"})
	t4 := NewTask("t4", []string{"t3", "t2"})
	t5 := NewTask("t5", []string{"t3"})
}
