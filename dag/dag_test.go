// This file is part of go-getoptions.
//
// Copyright (C) 2015-2024  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dag

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

func setupLogging() *bytes.Buffer {
	s := ""
	buf := bytes.NewBufferString(s)
	Logger.SetOutput(buf)
	return buf
}

func TestDag(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error

	sm := sync.Mutex{}
	results := []int{}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			sm.Lock()
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

	g := NewGraph("test graph")
	g.TaskDependensOn(tm.Get("t1"), tm.Get("t2"), tm.Get("t3"))
	g.TaskDependensOn(tm.Get("t2"), tm.Get("t4"))
	g.TaskDependensOn(tm.Get("t3"), tm.Get("t4"))
	g.TaskDependensOn(tm.Get("t4"), tm.Get("t5"))
	g.TaskDependensOn(tm.Get("t6"), tm.Get("t2"))
	g.TaskDependensOn(tm.Get("t6"), tm.Get("t8"))
	g.TaskDependensOn(tm.Get("t7"), tm.Get("t5"))

	// Validate before running
	err = g.Validate(tm)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	_, err = g.DepthFirstSort()
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.Run(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	before := func(s []int, a, b int) bool {
		ai, bi := -1, -1
		for i, e := range s {
			if e == a {
				ai = i
			}
			if e == b {
				bi = i
			}
		}
		return ai < bi
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
		case 4, 5, 6, 7:
			if e != 2 && e != 3 && e != 6 && e != 1 {
				t.Errorf("Wrong list: %v\n", results)
			}
		}
	}
	if !before(results, 2, 1) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 3, 1) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 4, 2) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 4, 3) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 5, 4) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 2, 6) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 8, 6) {
		t.Errorf("Wrong list: %v\n", results)
	}
	if !before(results, 5, 7) {
		t.Errorf("Wrong list: %v\n", results)
	}
	expectedDiagram := `
digraph G {
	label = "test graph";
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

	// Empty graph
	g2 := NewGraph("test graph 2")
	err = g2.Validate(nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
	err = g2.Run(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
}

func TestRunErrorCollection(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}

	g := NewGraph("test graph")
	g.AddTask(NewTask("t1", generateFn(1)))
	g.AddTask(NewTask("t2", generateFn(2)))
	g.AddTask(NewTask("t3", generateFn(3)))
	g.AddTask(NewTask("", generateFn(4)))   // ErrorTaskID
	g.AddTask(NewTask("t5", nil))           // ErrorTaskFn
	g.AddTask(nil)                          // ErrorTaskNil
	g.AddTask(NewTask("", generateFn(123))) // ErrorTaskID
	g.AddTask(NewTask("123", nil))          // ErrorTaskFn

	g.TaskDependensOn(g.Task("t2"), g.Task("t1"))
	g.TaskDependensOn(g.Task("t3"), g.Task("t2"))
	g.TaskDependensOn(g.Task("t3"), g.Task("t0")) // ErrorTaskNotFound, ErrorTaskFn
	g.TaskDependensOn(g.Task("t0"), g.Task("t3")) // ErrorTaskNotFound, ErrorTaskFn
	g.TaskDependensOn(g.Task("t2"), g.Task("t1")) // ErrorTaskDependencyDuplicate
	g.TaskDependensOn(g.Task("t2"), nil)          // ErrorTaskNil

	err = g.Validate(nil)
	var errs *Errors
	if err == nil || !errors.As(err, &errs) {
		t.Fatalf("Unexpected error: %s\n", err)
	}
	if len(errs.Errors) != 11 {
		t.Fatalf("Unexpected error count: %d\n", len(errs.Errors))
	}
	if !errors.Is(errs.Errors[0], ErrorTaskID) {
		t.Fatalf("Unexpected error at %d, %s\n", 0, errs.Error())
	}
	if !errors.Is(errs.Errors[1], ErrorTaskFn) {
		t.Fatalf("Unexpected error at %d, %s\n", 1, errs.Error())
	}
	if !errors.Is(errs.Errors[2], ErrorTaskNil) {
		t.Fatalf("Unexpected error at %d, %s\n", 2, errs.Error())
	}
	if !errors.Is(errs.Errors[3], ErrorTaskID) {
		t.Fatalf("Unexpected error at %d, %s\n", 3, errs.Error())
	}
	if !errors.Is(errs.Errors[4], ErrorTaskFn) {
		t.Fatalf("Unexpected error at %d, %s\n", 4, errs.Error())
	}
	if !errors.Is(errs.Errors[5], ErrorTaskNotFound) {
		t.Fatalf("Unexpected error at %d, %s\n", 5, errs.Error())
	}
	if !errors.Is(errs.Errors[6], ErrorTaskFn) {
		t.Fatalf("Unexpected error at %d, %s\n", 6, errs.Error())
	}
	if !errors.Is(errs.Errors[7], ErrorTaskNotFound) {
		t.Fatalf("Unexpected error at %d, %s\n", 7, errs.Error())
	}
	if !errors.Is(errs.Errors[8], ErrorTaskFn) {
		t.Fatalf("Unexpected error at %d, %s\n", 8, errs.Error())
	}
	if !errors.Is(errs.Errors[9], ErrorTaskDependencyDuplicate) {
		t.Fatalf("Unexpected error at %d, %s\n", 9, errs.Error())
	}
	if !errors.Is(errs.Errors[10], ErrorTaskNil) {
		t.Fatalf("Unexpected error at %d, %s\n", 10, errs.Error())
	}
	expected := `test graph errors found:
> missing task ID
> missing task function for t5
> nil task given
> missing task ID
> missing task function for 123
> task not found: t0
> missing task function for t0
> task not found: t0
> missing task function for t0
> task dependency already defined: t2 -> t1
> nil task given`
	if expected != errs.Error() {
		t.Fatalf("Unexpected error: '%s'\nexpected: '%s'\n", err, expected)
	}

	err = g.Run(context.Background(), nil, nil)
	if err == nil {
		t.Errorf("Wrong error: %s\n", err)
	}
}

func TestCycle(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			return nil
		}
	}

	g := NewGraph("test graph")
	g.AddTask(NewTask("t1", generateFn(1)))
	g.AddTask(NewTask("t2", generateFn(2)))

	g.TaskDependensOn(g.Task("t1"), g.Task("t2"))
	g.TaskDependensOn(g.Task("t2"), g.Task("t1"))
	_, err := g.DepthFirstSort()
	if err == nil || !errors.Is(err, ErrorGraphHasCycle) {
		t.Errorf("Wrong error: %s\n", err)
	}

	err = g.Run(context.Background(), nil, nil)
	if err == nil || !errors.Is(err, ErrorGraphHasCycle) {
		t.Errorf("Wrong error: %s\n", err)
	}
}

func TestDagTaskError(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error

	sm := sync.Mutex{}
	results := []int{}
	g := NewGraph("test graph")
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			time.Sleep(30 * time.Millisecond)
			if n == 4 {
				return fmt.Errorf("failure reason")
			}
			sm.Lock()
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

	err = g.Run(context.Background(), nil, nil)
	var errs *Errors
	if err == nil || !errors.As(err, &errs) {
		t.Fatalf("Unexpected error: %s\n", err)
	}
	if len(errs.Errors) != 5 {
		t.Fatalf("Unexpected error size, %d: %s\n", len(errs.Errors), err)
	}
	if errs.Errors[0].Error() != "Task test graph:t4 error: failure reason" {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[0])
	}
	if !errors.Is(errs.Errors[1], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if !errors.Is(errs.Errors[2], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[2])
	}
	if !errors.Is(errs.Errors[3], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[3])
	}
	if !errors.Is(errs.Errors[4], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[4])
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

func TestDagContexDone(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error
	ctx, cancel := context.WithCancel(context.Background())

	sm := sync.Mutex{}
	results := []int{}
	g := NewGraph("test graph")
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			sm.Lock()
			results = append(results, n)
			sm.Unlock()
			if n == 4 {
				cancel()
			}
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

	err = g.Run(ctx, nil, nil)
	var errs *Errors
	if err == nil || !errors.As(err, &errs) {
		t.Fatalf("Unexpected error: %s\n", err)
	}
	if len(errs.Errors) != 5 {
		t.Fatalf("Unexpected error size, %d: %s\n", len(errs.Errors), err)
	}
	if errs.Errors[0].Error() != "cancelation received or time out reached" {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[0])
	}
	if !errors.Is(errs.Errors[1], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if !errors.Is(errs.Errors[2], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if !errors.Is(errs.Errors[3], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if !errors.Is(errs.Errors[4], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if len(results) > 4 {
		t.Errorf("Wrong list: %v\n", results)
	}
	for i, e := range results {
		switch i {
		case 0, 1:
			if e != 5 && e != 8 {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 2, 3:
			if e != 7 && e != 4 {
				t.Errorf("Wrong list: %v\n", results)
			}
		}
	}
}

func TestDagTaskSkipParents(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })
	var err error

	sm := sync.Mutex{}
	results := []int{}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			sm.Lock()
			if n == 4 {
				sm.Unlock()
				return ErrorSkipParents
			}
			results = append(results, n)
			sm.Unlock()
			return nil
		}
	}

	g := NewGraph("test graph")
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

	err = g.Run(context.Background(), nil, nil)
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

func TestDagSerial(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error

	results := []int{}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			for i := 0; i < 5; i++ {
				t.Logf("Task %d", n)
			}
			results = append(results, n)
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

	g := NewGraph("test graph").SetSerial()
	g.TaskDependensOn(tm.Get("t1"), tm.Get("t2"), tm.Get("t3"))
	g.TaskDependensOn(tm.Get("t2"), tm.Get("t4"))
	g.TaskDependensOn(tm.Get("t3"), tm.Get("t4"))
	g.TaskDependensOn(tm.Get("t4"), tm.Get("t5"))
	g.TaskDependensOn(tm.Get("t6"), tm.Get("t2"))
	g.TaskDependensOn(tm.Get("t6"), tm.Get("t8"))
	g.TaskDependensOn(tm.Get("t7"), tm.Get("t5"))

	// Validate before running
	err = g.Validate(tm)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.Run(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
	seen := map[int]bool{}
	for _, e := range results {
		seen[e] = true
		switch e {
		case 5:
			if _, ok := seen[4]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
			if _, ok := seen[7]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 4:
			if _, ok := seen[3]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
			if _, ok := seen[2]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 2:
			if _, ok := seen[1]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
			if _, ok := seen[6]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 3:
			if _, ok := seen[1]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
		case 8:
			if _, ok := seen[6]; ok {
				t.Errorf("Wrong list: %v\n", results)
			}
		}
	}
}

func TestTaskMap(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })
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
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })
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

	g := NewGraph("graph")
	err = g.Validate(tm)
	if err == nil || !errors.As(err, &errs) {
		t.Fatalf("Unexpected error: %s\n", err)
	}
	if len(errs.Errors) != 3 {
		t.Fatalf("Unexpected error: %s\n", errs)
	}

	if tm.Get("t5").ID != "t5" {
		t.Errorf("Wrong value: %s\n", tm.Get("t5").ID)
	}
}

func TestMaxParallel(t *testing.T) {
	tests := []struct {
		concurrency int
		expected    int
	}{
		{0, 16},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 4},
		{5, 5},
		{6, 6},
		{7, 7},
		{8, 8},
		{9, 9},
		{10, 10},
		{11, 11},
		{12, 12},
		{13, 13},
		{14, 14},
		{15, 15},
		{16, 16},
	}
	for _, test := range tests {
		t.Run(fmt.Sprintf("%d", test.concurrency), func(t *testing.T) {
			buf := setupLogging()
			t.Cleanup(func() { t.Log(buf.String()) })
			Logger.SetOutput(buf)

			var err error
			sm := sync.Mutex{}
			workMutex := sync.Mutex{}
			peakConcurrency := 0
			currentConcurrency := 0

			generateFn := func(n int) getoptions.CommandFn {
				return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
					sm.Lock()
					currentConcurrency += 1
					if currentConcurrency > peakConcurrency {
						peakConcurrency = currentConcurrency
					}
					sm.Unlock()
					workMutex.Lock()
					time.Sleep(3 * time.Millisecond)
					workMutex.Unlock()
					sm.Lock()
					currentConcurrency -= 1
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
			tm.Add("t9", generateFn(9))
			tm.Add("t10", generateFn(10))
			tm.Add("t11", generateFn(11))
			tm.Add("t12", generateFn(12))
			tm.Add("t13", generateFn(13))
			tm.Add("t14", generateFn(14))
			tm.Add("t15", generateFn(15))
			tm.Add("t16", generateFn(16))

			g := NewGraph("test graph")
			g.SetMaxParallel(test.concurrency)
			g.AddTask(tm.Get("t1"))
			g.AddTask(tm.Get("t2"))
			g.AddTask(tm.Get("t3"))
			g.AddTask(tm.Get("t4"))
			g.AddTask(tm.Get("t5"))
			g.AddTask(tm.Get("t6"))
			g.AddTask(tm.Get("t7"))
			g.AddTask(tm.Get("t8"))
			g.AddTask(tm.Get("t9"))
			g.AddTask(tm.Get("t10"))
			g.AddTask(tm.Get("t11"))
			g.AddTask(tm.Get("t12"))
			g.AddTask(tm.Get("t13"))
			g.AddTask(tm.Get("t14"))
			g.AddTask(tm.Get("t15"))
			g.AddTask(tm.Get("t16"))

			// Start a goroutine and keep the mutex locked for enough time for all
			// routines to reach it then release the lock.
			workMutex.Lock()
			go func() {
				time.Sleep(70 * time.Millisecond)
				workMutex.Unlock()
			}()

			err = g.Run(context.Background(), nil, nil)
			if err != nil {
				t.Errorf("Unexpected error: %s\n", err)
			}
			if peakConcurrency != test.expected {
				t.Errorf("wrong peak concurrency %d", peakConcurrency)
			}
		})
	}
}

func TestBufferedOutput(t *testing.T) {
	buf := setupLogging()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error

	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			stdout := Stdout(ctx)
			stderr := Stderr(ctx)
			for i := 0; i < 10; i++ {
				fmt.Fprintf(stdout, "Running Fn %d\n", n)
				fmt.Fprintf(stderr, "Running Fn %d\n", n)
			}
			return nil
		}
	}

	tm := NewTaskMap()
	tm.Add("t1", generateFn(1))
	tm.Add("t2", generateFn(2))
	tm.Add("t3", generateFn(3))
	tm.Add("t4", generateFn(4))
	tm.Add("t5", generateFn(5))

	s := ""
	outBuf := bytes.NewBufferString(s)
	g := NewGraph("test graph")
	g.SetOutputBuffer(outBuf)
	g.AddTask(tm.Get("t1"))
	g.AddTask(tm.Get("t2"))
	g.AddTask(tm.Get("t3"))
	g.AddTask(tm.Get("t4"))
	g.AddTask(tm.Get("t5"))

	err = g.Run(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	// Even though things run in parallel the output needs to be the same in groups of 20 lines.
	lines := strings.Split(outBuf.String(), "\n")
	for i := range lines {
		if i+1 == len(lines) {
			break
		}
		if (i+1)%20 != 0 && lines[i] != lines[i+1] {
			t.Errorf("wrong output idx %d: %s != %s\n", i, lines[i], lines[i+1])
		}
	}

	generateTestFn := func(t *testing.T, n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			stdout := Stdout(ctx)
			stderr := Stderr(ctx)
			if stdout != os.Stdout {
				t.Errorf("Invalid stdout\n")
			}
			if stderr != os.Stderr {
				t.Errorf("Invalid stdout\n")
			}
			return nil
		}
	}

	tm = NewTaskMap()
	tm.Add("t1", generateTestFn(t, 1))
	tm.Add("t2", generateTestFn(t, 2))

	g = NewGraph("test graph")
	g.AddTask(tm.Get("t1"))
	g.AddTask(tm.Get("t2"))

	err = g.Run(context.Background(), nil, nil)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
}
