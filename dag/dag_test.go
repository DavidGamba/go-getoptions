package dag

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/DavidGamba/go-getoptions"
)

func TestDag(t *testing.T) {
	var err error
	results := []int{}
	g := NewGraph()
	addTask := func(t *Task) {
		if err != nil {
			return
		}
		err = g.AddTask(t)
	}
	generateTask := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			results = append(results, n)
			return nil
		}
	}
	addTask(NewTask("t1", generateTask(1)))
	addTask(NewTask("t2", generateTask(2)))
	addTask(NewTask("t3", generateTask(3)))
	addTask(NewTask("t4", generateTask(4)))
	addTask(NewTask("t5", generateTask(5)))
	addTask(NewTask("t6", generateTask(6)))
	addTask(NewTask("t7", generateTask(7)))
	addTask(NewTask("t8", generateTask(8)))
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.AddTask(NewTask("t3", generateTask(3)))
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskDuplicate) {
		t.Errorf("Wrong error: %s\n", err)
	}

	g.TaskDependensOn("t1", "t2", "t3")
	g.TaskDependensOn("t2", "t4")
	g.TaskDependensOn("t3", "t4")
	g.TaskDependensOn("t4", "t5")
	g.TaskDependensOn("t6", "t2")
	g.TaskDependensOn("t6", "t8")
	err = g.TaskDependensOn("t7", "t5")
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.TaskDependensOn("t4", "t0")
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskNotFound) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.TaskDependensOn("t0", "t5")
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskNotFound) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.TaskDependensOn("t4", "t5")
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskDependencyDuplicate) {
		t.Errorf("Wrong error: %s\n", err)
	}

	start, err := g.GetStartVertex()
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
	if len(start) != 2 || (!reflect.DeepEqual(GetVertexIDs(start), []ID{"t5", "t8"}) &&
		!reflect.DeepEqual(GetVertexIDs(start), []ID{"t8", "t5"})) {
		t.Errorf("Wrong start list: %v\n", GetVertexIDs(start))
	}
}
