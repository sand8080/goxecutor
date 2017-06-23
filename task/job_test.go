package task

import (
	"fmt"
	"github.com/cznic/mathutil"
	"sort"
	"testing"
)

func TestJobsBuilder_AddTask(t *testing.T) {
	tasks := map[string]*Task{
		"t1": NewTask("t1", []string{}),
		"t2": NewTask("t2", []string{"t1"}),
		"t3": NewTask("t3", []string{"t1"}),
		"t4": NewTask("t4", []string{"t3", "t2"}),
		"t5": NewTask("t5", []string{"t3"}),
	}

	taskNames := sort.StringSlice{"t1", "t2", "t3", "t4", "t5"}
	mathutil.PermutationFirst(taskNames)
	hasNext := true
	for hasNext {
		builder := NewJobsBuilder()
		for _, name := range taskNames {
			t := tasks[name]
			builder.AddTask(t, false)
		}
		fmt.Println(builder)
		hasNext = mathutil.PermutationNext(taskNames)
	}
}
