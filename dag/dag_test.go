package dag

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

func TestDag(t *testing.T) {
	var err error

	sm := sync.Mutex{}
	results := []int{}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			sm.Lock()
			time.Sleep(1 * time.Millisecond)
			results = append(results, n)
			sm.Unlock()
			return nil
		}
	}

	tm := NewTaskMap()
	tm.Add("t1", generateFn(1))
	tm.Add("t2", generateFn(2))
	tm.Add("t3", generateFn(3))
	tm.Add("t4", generateFn(4))
	tm.Add("t5", generateFn(5))
	tm.Add("t6", generateFn(6))
	tm.Add("t7", generateFn(7))
	tm.Add("t8", generateFn(8))
	err = tm.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	g := NewGraph()
	g.TaskDependensOn(tm.Get("t1"), tm.Get("t2"), tm.Get("t3"))
	g.TaskDependensOn(tm.Get("t2"), tm.Get("t4"))
	g.TaskDependensOn(tm.Get("t3"), tm.Get("t4"))
	g.TaskDependensOn(tm.Get("t4"), tm.Get("t5"))
	g.TaskDependensOn(tm.Get("t6"), tm.Get("t2"))
	g.TaskDependensOn(tm.Get("t6"), tm.Get("t8"))
	err = g.TaskDependensOn(tm.Get("t7"), tm.Get("t5"))
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
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
	"t1";
	"t2";
	"t1" -> "t2";
	"t3";
	"t1" -> "t3";
	"t4";
	"t2" -> "t4";
	"t3" -> "t4";
	"t5";
	"t4" -> "t5";
	"t6";
	"t6" -> "t2";
	"t8";
	"t6" -> "t8";
	"t7";
	"t7" -> "t5";
}`
	if g.String() != expectedDiagram {
		t.Errorf("Wrong output: '%s'\nexpected: '%s'\n", g.String(), expectedDiagram)
	}
}

func TestRunErrorCollection(t *testing.T) {
	var err error
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}

	g := NewGraph()
	g.AddTask(NewTask("t1", generateFn(1)))
	g.AddTask(NewTask("t2", generateFn(2)))
	g.AddTask(NewTask("t3", generateFn(3)))

	g.AddTask(NewTask("", generateFn(4)))

	g.AddTask(NewTask("t5", nil))

	g.AddTask(nil)

	g.AddTask(NewTask("", generateFn(123)))

	g.AddTask(NewTask("123", nil))

	g.TaskDependensOn(g.Task("t2"), g.Task("t1"))
	err = g.TaskDependensOn(g.Task("t3"), g.Task("t2"))
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.TaskDependensOn(g.Task("t3"), g.Task("t0"))
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskNotFound) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.TaskDependensOn(g.Task("t0"), g.Task("t3"))
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskNotFound) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.TaskDependensOn(g.Task("t2"), g.Task("t1"))
	if err == nil {
		t.Errorf("Expected error none triggered\n")
	}
	if !errors.Is(err, ErrorTaskDependencyDuplicate) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.Run(nil, nil, nil)
	if err == nil || !errors.Is(err, ErrorGraph) {
		t.Errorf("Wrong error: %s\n", err)
	}
}

func TestCycle(t *testing.T) {
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}

	g := NewGraph()
	g.AddTask(NewTask("t1", generateFn(1)))
	g.AddTask(NewTask("t2", generateFn(2)))

	g.TaskDependensOn(g.Task("t1"), g.Task("t2"))
	g.TaskDependensOn(g.Task("t2"), g.Task("t1"))
	_, err := g.DephFirstSort()
	if err == nil || !errors.Is(err, ErrorGraphHasCycle) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.Run(nil, nil, nil)
	if err == nil || !errors.Is(err, ErrorGraphHasCycle) {
		t.Errorf("Wrong error: %s\n", err)
	}
}

func TestDagTaskError(t *testing.T) {
	var err error

	sm := sync.Mutex{}
	results := []int{}
	g := NewGraph()
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			if n == 4 {
				return fmt.Errorf("error for %d", n)
			}
			sm.Lock()
			time.Sleep(1 * time.Millisecond)
			results = append(results, n)
			sm.Unlock()
			return nil
		}
	}
	g.AddTask(NewTask("t1", generateFn(1)))
	g.AddTask(NewTask("t2", generateFn(2)))
	g.AddTask(NewTask("t3", generateFn(3)))
	g.AddTask(NewTask("t4", generateFn(4)))
	g.AddTask(NewTask("t5", generateFn(5)))
	g.AddTask(NewTask("t6", generateFn(6)))
	g.AddTask(NewTask("t7", generateFn(7)))
	g.AddTask(NewTask("t8", generateFn(8)))

	g.TaskDependensOn(g.Task("t1"), g.Task("t2"), g.Task("t3"))
	g.TaskDependensOn(g.Task("t2"), g.Task("t4"))
	g.TaskDependensOn(g.Task("t3"), g.Task("t4"))
	g.TaskDependensOn(g.Task("t4"), g.Task("t5"))
	g.TaskDependensOn(g.Task("t6"), g.Task("t2"))
	g.TaskDependensOn(g.Task("t6"), g.Task("t8"))
	g.TaskDependensOn(g.Task("t7"), g.Task("t5"))

	err = g.Run(nil, nil, nil)
	if err == nil || !errors.Is(err, ErrorRunTask) {
		t.Errorf("Wrong error: %s\n", err)
	}
	if len(results) > 3 {
		t.Errorf("Wrong list: %v\n", results)
	}
	for i, e := range results {
		switch i {
		case 0, 1:
			if e != 5 && e != 8 {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 2:
			if e != 7 {
				t.Errorf("Wrong list: %v\n", results)
			}
		}
	}
}

func TestDagTaskSkipParents(t *testing.T) {
	var err error

	sm := sync.Mutex{}
	results := []int{}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			sm.Lock()
			time.Sleep(1 * time.Millisecond)
			if n == 4 {
				sm.Unlock()
				return ErrorSkipParents
			}
			results = append(results, n)
			sm.Unlock()
			return nil
		}
	}

	g := NewGraph()
	g.AddTask(NewTask("t1", generateFn(1)))
	g.AddTask(NewTask("t2", generateFn(2)))
	g.AddTask(NewTask("t3", generateFn(3)))
	g.AddTask(NewTask("t4", generateFn(4)))
	g.AddTask(NewTask("t5", generateFn(5)))
	g.AddTask(NewTask("t6", generateFn(6)))
	g.AddTask(NewTask("t7", generateFn(7)))
	g.AddTask(NewTask("t8", generateFn(8)))

	g.TaskDependensOn(g.Task("t1"), g.Task("t2"), g.Task("t3"))
	g.TaskDependensOn(g.Task("t2"), g.Task("t4"))
	g.TaskDependensOn(g.Task("t3"), g.Task("t4"))
	g.TaskDependensOn(g.Task("t4"), g.Task("t5"))
	g.TaskDependensOn(g.Task("t6"), g.Task("t2"))
	g.TaskDependensOn(g.Task("t6"), g.Task("t8"))
	g.TaskDependensOn(g.Task("t7"), g.Task("t5"))

	err = g.Run(nil, nil, nil)
	if err != nil {
		t.Errorf("Wrong error: %s\n", err)
	}
	if len(results) > 3 {
		t.Errorf("Wrong list: %v\n", results)
	}
	for i, e := range results {
		switch i {
		case 0, 1:
			if e != 5 && e != 8 {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 2:
			if e != 7 {
				t.Errorf("Wrong list: %v\n", results)
			}
		}
	}
}

func TestTaskMap(t *testing.T) {
	var err error
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}

	tm := NewTaskMap()
	tm.Add("t1", generateFn(1))
	tm.Add("t2", generateFn(2))
	tm.Add("t3", generateFn(3))

	err = tm.Validate()
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	if tm.Get("t3").ID != "t3" {
		t.Errorf("Wrong value: %s\n", tm.Get("t3").ID)
	}
}

func TestTaskMapErrors(t *testing.T) {
	var err error
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}

	tm := NewTaskMap()
	tm.Add("t1", generateFn(1))
	tm.Add("t2", generateFn(2))
	tm.Add("t2", generateFn(2)) // ErrorTaskDuplicate
	tm.Add("", generateFn(3))   // ErrorTaskID
	tm.Add("t4", nil)           // ErrorTaskFn

	err = tm.Validate()
	var errs *Errors
	if err == nil || !errors.As(err, &errs) {
		t.Fatalf("Unexpected error: %s\n", err)
	}
	if len(errs.Errors) != 3 {
		t.Fatalf("Unexpected error: %#v\n", errs.Errors)
	}
	if !errors.Is(errs.Errors[0], ErrorTaskDuplicate) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[0])
	}
	if !errors.Is(errs.Errors[1], ErrorTaskID) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if !errors.Is(errs.Errors[2], ErrorTaskFn) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[2])
	}

	if tm.Get("t5").ID != "t5" {
		t.Errorf("Wrong value: %s\n", tm.Get("t5").ID)
	}
}
