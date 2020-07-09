package dag

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

func TestDag(t *testing.T) {
	var err error

	sm := sync.Mutex{}
	results := []int{}
	g := NewGraph()
	addTask := func(id string, fn getoptions.CommandFn) {
		if err != nil {
			return
		}
		err = g.CreateTask(id, fn)
	}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			sm.Lock()
			time.Sleep(1 * time.Millisecond)
			results = append(results, n)
			sm.Unlock()
			return nil
		}
	}
	addTask("t1", generateFn(1))
	addTask("t2", generateFn(2))
	addTask("t3", generateFn(3))
	addTask("t4", generateFn(4))
	addTask("t5", generateFn(5))
	addTask("t6", generateFn(6))
	addTask("t7", generateFn(7))
	addTask("t8", generateFn(8))
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.AddTask(NewTask("t3", generateFn(3)))
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

	_, err = g.DephFirstSort()
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.Run(nil, nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
	for i, e := range results {
		switch i {
		case 0, 1:
			if e != 5 && e != 8 {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 2, 3:
			if e != 4 && e != 7 {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 4, 5:
			if e != 2 && e != 3 {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 6, 7:
			if e != 1 && e != 6 {
				t.Errorf("Wrong list: %v\n", results)
			}
		}
	}
	expectedDiagram := `
digraph G {
	rankdir = TB;
	"t1" -> "t2";
	"t1" -> "t3";
	"t2" -> "t4";
	"t3" -> "t4";
	"t4" -> "t5";
	"t6" -> "t2";
	"t6" -> "t8";
	"t7" -> "t5";
}`
	if g.String() != expectedDiagram {
		t.Errorf("Wrong output: '%s'\nexpected: '%s'\n", g.String(), expectedDiagram)
	}
}

func TestCycle(t *testing.T) {
	var err error
	g := NewGraph()
	addTask := func(id string, fn getoptions.CommandFn) {
		if err != nil {
			return
		}
		err = g.CreateTask(id, fn)
	}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}
	addTask("t1", generateFn(1))
	addTask("t2", generateFn(2))

	g.TaskDependensOn("t1", "t2")
	g.TaskDependensOn("t2", "t1")
	_, err = g.DephFirstSort()
	if err == nil || !errors.Is(err, ErrorGraphHasCycle) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.Run(nil, nil, nil)
	if err == nil || !errors.Is(err, ErrorGraphHasCycle) {
		t.Errorf("Wrong error: %s\n", err)
	}
}
