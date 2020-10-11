// This file is part of go-getoptions.
//
// Copyright (C) 2015-2020  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// TODO: Allow defining an Entrypoint into the graph and operate from that point.
// Subgraph concept.
// TODO: Aggregate logs into a buffer and print the logs when the task completes so they aren't merged together when running in parallel?
// TODO: Pass context to subtask.
// TODO: Decide if t.Cleanup is reason to go 1.14+
// Requires Go 1.13+

/*
Package dag - Lightweight Directed Acyclic Graph (DAG) Build System.

It allows building a list of tasks and then running the tasks in different DAG trees.

The tree dependencies are calculated and tasks that have met their dependencies are run in parallel.
There is an option to run it serially for cases where user interaction is required.
*/
package dag

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags)

type (
	ID string

	Task struct {
		ID ID
		Fn getoptions.CommandFn
		sm sync.Mutex
	}

	TaskMap struct {
		m    map[string]*Task
		errs *Errors
	}

	Errors struct {
		Msg    string
		Errors []error
	}

	Vertex struct {
		ID       ID
		Task     *Task
		Children []*Vertex
		Parents  []*Vertex
		status   runStatus
	}

	Graph struct {
		Name           string
		TickerDuration time.Duration
		Vertices       map[ID]*Vertex
		dotDiagram     string
		errs           *Errors
		serial         bool
	}

	runStatus int

	visitStatus int
)

const (
	runPending runStatus = iota
	runInProgress
	runSkip
	runDone
)

const (
	// Vertex hasn't been visited
	unvisited visitStatus = iota
	// Vertex has been seen but not traversed
	visited
	// Vertex children have been traversed
	traversed
)

var ErrorTaskNil = fmt.Errorf("nil task given")
var ErrorTaskID = fmt.Errorf("missing task ID")
var ErrorTaskFn = fmt.Errorf("missing task function")
var ErrorTaskDuplicate = fmt.Errorf("task definition already exists")
var ErrorTaskNotFound = fmt.Errorf("task not found")
var ErrorTaskDependencyDuplicate = fmt.Errorf("task dependency already defined")
var ErrorGraphHasCycle = fmt.Errorf("graph has a cycle")
var ErrorTaskSkipped = fmt.Errorf("skipped")

// ErrorSkipParents - Allows for conditional tasks that allow a task to Skip all parent tasks without failing the run
var ErrorSkipParents = fmt.Errorf("skip parents without failing")

func (errs *Errors) Error() string {
	msg := ""
	for _, e := range errs.Errors {
		msg += fmt.Sprintf("\n> %s", e)
	}
	return fmt.Sprintf("%s errors found:%s", errs.Msg, msg)
}

// NewTask - Allows creating a reusable Task that can be passed to multiple graphs.
func NewTask(id string, fn getoptions.CommandFn) *Task {
	return &Task{
		ID: ID(id),
		Fn: fn,
		sm: sync.Mutex{},
	}
}

func (t *Task) Lock() {
	t.sm.Lock()
}

func (t *Task) Unlock() {
	t.sm.Unlock()
}

// NewTaskMap - Map helper to group multiple Tasks.
func NewTaskMap() *TaskMap {
	return &TaskMap{
		m:    make(map[string]*Task),
		errs: &Errors{"Task Map", make([]error, 0)},
	}
}

// Add - Adds a new task to the TaskMap.
// Errors collected for TaskMap.Validate().
func (tm *TaskMap) Add(id string, fn getoptions.CommandFn) *Task {
	if id == "" {
		err := ErrorTaskID
		tm.errs.Errors = append(tm.errs.Errors, err)
	}
	if fn == nil {
		err := fmt.Errorf("%w for %s", ErrorTaskFn, id)
		tm.errs.Errors = append(tm.errs.Errors, err)
	}
	if _, ok := tm.m[id]; ok {
		err := fmt.Errorf("%w: '%s'", ErrorTaskDuplicate, id)
		tm.errs.Errors = append(tm.errs.Errors, err)
	}
	newTask := NewTask(id, fn)
	tm.m[id] = newTask
	return newTask
}

// Get - Gets the task from the TaskMap, if not found returns a new empty Task.
// Errors collected for TaskMap.Validate().
func (tm *TaskMap) Get(id string) *Task {
	if t, ok := tm.m[id]; ok {
		return t
	}
	err := fmt.Errorf("%w: %s", ErrorTaskNotFound, id)
	tm.errs.Errors = append(tm.errs.Errors, err)
	return NewTask(id, nil)
}

// Validate - Verifies that there are no errors in the TaskMap.
func (tm *TaskMap) Validate() error {
	if len(tm.errs.Errors) != 0 {
		return tm.errs
	}
	return nil
}

// NewGraph - Create a graph that allows running Tasks in parallel.
func NewGraph(name string) *Graph {
	return &Graph{
		Name:           name,
		TickerDuration: 1 * time.Millisecond,
		Vertices:       make(map[ID]*Vertex),
		errs:           &Errors{name, make([]error, 0)},
	}
}

// SetSerial - The call to Run() will run the Tasks serially.
// Useful when the tasks require user input and the user needs to see logs in order to make a decision.
func (g *Graph) SetSerial() *Graph {
	g.serial = true
	return g
}

// String - Returns a dot diagram of the graph.
func (g *Graph) String() string {
	dotDiagramHeader := fmt.Sprintf(`
digraph G {
	label = "%s";
	rankdir = TB;
`, g.Name)

	dotDiagramFooter := `}`

	return dotDiagramHeader + g.dotDiagram + dotDiagramFooter
}

// Task - Retrieve a Task from the graph by its ID.
func (g *Graph) Task(id string) *Task {
	vertex, ok := g.Vertices[ID(id)]
	if !ok {
		err := fmt.Errorf("%w: %s", ErrorTaskNotFound, id)
		g.errs.Errors = append(g.errs.Errors, err)
		// Return an empty task so downstream errors can reference the ID.
		return &Task{
			ID: ID(id),
			Fn: nil,
			sm: sync.Mutex{},
		}
	}
	return vertex.Task
}

// AddTask - Add Task to graph.
// Errors collected for Graph.Validate().
func (g *Graph) AddTask(t *Task) {
	err := g.addTask(t)
	if err != nil {
		g.errs.Errors = append(g.errs.Errors, err)
	}
}

func (g *Graph) addTask(t *Task) error {
	if t == nil {
		err := ErrorTaskNil
		return err
	}
	if t.ID == "" {
		err := ErrorTaskID
		return err
	}
	if t.Fn == nil {
		err := fmt.Errorf("%w for %s", ErrorTaskFn, t.ID)
		return err
	}
	// Allow duplicate definitions, use a Task Map if you want to ensure you define your tasks only once.
	// if _, ok := g.Vertices[t.ID]; ok {
	// 	err := fmt.Errorf("%w: %s", ErrorTaskDuplicate, t.ID)
	// 	g.errs = append(g.errs, err)
	// 	return err
	// }
	if _, ok := g.Vertices[t.ID]; !ok {
		g.dotDiagram += fmt.Sprintf("\t\"%s\";\n", t.ID)
	}
	g.Vertices[t.ID] = &Vertex{
		ID:       t.ID,
		Task:     t,
		Children: make([]*Vertex, 0),
		Parents:  make([]*Vertex, 0),
		status:   runPending,
	}
	return nil
}

func (g *Graph) retrieveOrAddVertex(t *Task) (*Vertex, error) {
	if t == nil {
		return nil, ErrorTaskNil
	}
	vertex, ok := g.Vertices[t.ID]
	if !ok {
		err := g.addTask(t)
		if err != nil {
			return nil, err
		}
		vertex = g.Vertices[t.ID]
	}
	return vertex, nil
}

// TaskDependensOn - Allows adding tasks to the graph and defining their edges.
func (g *Graph) TaskDependensOn(t *Task, tDependencies ...*Task) {
	vertex, err := g.retrieveOrAddVertex(t)
	if err != nil {
		g.errs.Errors = append(g.errs.Errors, err)
		return
	}
	for _, tDependency := range tDependencies {
		vDependency, err := g.retrieveOrAddVertex(tDependency)
		if err != nil {
			g.errs.Errors = append(g.errs.Errors, err)
			return
		}
		for _, c := range vertex.Children {
			if c.ID == vDependency.ID {
				err := fmt.Errorf("%w: %s -> %s", ErrorTaskDependencyDuplicate, vertex.ID, vDependency.ID)
				g.errs.Errors = append(g.errs.Errors, err)
				return
			}
		}
		g.dotDiagram += fmt.Sprintf("\t\"%s\" -> \"%s\";\n", vertex.ID, vDependency.ID)
		vertex.Children = append(vertex.Children, vDependency)
		vDependency.Parents = append(vDependency.Parents, vertex)
	}
}

// Validate - Verifies that there are no errors in the Graph.
// It also runs Validate() on the given TaskMap (pass nil if a TaskMap wasn't used).
func (g *Graph) Validate(tm *TaskMap) error {
	if tm != nil {
		err := tm.Validate()
		if err != nil {
			return err
		}
	}
	if len(g.errs.Errors) != 0 {
		return g.errs
	}
	return nil
}

// Run - Execute the graph tasks in parallel where possible.
// It checks for tasks updates every 1 Millisecond by default.
// Modify using the graph.TickerDuration
func (g *Graph) Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	runStart := time.Now()

	if len(g.errs.Errors) != 0 {
		return g.errs
	}

	if len(g.Vertices) == 0 {
		return nil
	}

	// Check for cycles
	_, err := g.DephFirstSort()
	if err != nil {
		return err
	}

	handledContext := false
	type IDErr struct {
		ID    ID
		Error error
	}
	done := make(chan IDErr)
LOOP:
	for {
		select {
		case iderr := <-done:
			g.Vertices[iderr.ID].status = runDone
			if iderr.Error != nil {
				err := fmt.Errorf("Task %s error: %w", iderr.ID, iderr.Error)
				Logger.Printf("%s\n", err)
				if !errors.Is(iderr.Error, ErrorSkipParents) {
					g.errs.Errors = append(g.errs.Errors, err)
					continue
				}
				skipParents(g.Vertices[iderr.ID])
			}
		default:
			v, allDone, ok := g.getNextVertex()
			if allDone {
				// Tasks completed outside of this task run.
				// For example when the same graph was passed to multiple methods and run multiple times.
				break LOOP
			}
			select {
			case <-ctx.Done():
				if handledContext {
					break
				}
				Logger.Printf("Cancelation received or time out reached, allowing in-progress tasks to finish, skipping the rest.\n")
				g.errs.Errors = append(g.errs.Errors, fmt.Errorf("cancelation received or time out reached"))
				handledContext = true
			default:
				break
			}
			if !ok {
				time.Sleep(g.TickerDuration)
				// TODO: Add a timeout to not wait infinitely for a task.
				continue
			}
			if v.status == runSkip {
				v.status = runInProgress
				Logger.Printf("Skipped Task %s\n", v.ID)
				go func(done chan IDErr, v *Vertex) {
					done <- IDErr{v.ID, nil}
				}(done, v)
				continue
			}
			v.status = runInProgress
			if len(g.errs.Errors) != 0 {
				go func(done chan IDErr, v *Vertex) {
					done <- IDErr{v.ID, ErrorTaskSkipped}
				}(done, v)
				continue
			}
			Logger.Printf("Running Task %s\n", v.ID)
			go func(done chan IDErr, v *Vertex) {
				start := time.Now()
				v.Task.Lock()
				defer v.Task.Unlock()
				err := v.Task.Fn(ctx, opt, args)
				Logger.Printf("Completed Task %s in %s\n", v.ID, durationStr(time.Since(start)))
				done <- IDErr{v.ID, err}
			}(done, v)
		}
	}
	Logger.Printf("Completed Run in %s\n", durationStr(time.Since(runStart)))

	if len(g.errs.Errors) != 0 {
		return g.errs
	}
	return nil
}

func durationStr(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%02dm:%02ds", m, s)
}

// getNextVertex - Walks the graph to get the next Vertex that is pending and without pending dependencies.
// Return: vertex, allDone, ok
func (g *Graph) getNextVertex() (*Vertex, bool, bool) {
	doneCount := 0
	if g.serial {
		for _, vertex := range g.Vertices {
			if vertex.status == runInProgress {
				return vertex, false, false
			}
			for _, child := range vertex.Children {
				if child.status == runInProgress {
					return child, false, false
				}
			}
		}
	}
	for _, vertex := range g.Vertices {
		if vertex.status != runPending && vertex.status != runSkip {
			if vertex.status == runDone {
				doneCount++
			}
			continue
		}
		if len(vertex.Children) == 0 {
			return vertex, false, true
		}
		childPending := false
		for _, child := range vertex.Children {
			if child.status == runPending {
				childPending = true
			}
			if child.status == runInProgress {
				childPending = true
			}
		}
		if !childPending {
			return vertex, false, true
		}
	}
	if doneCount == len(g.Vertices) {
		return nil, true, false
	}
	return nil, false, false
}

// skipParents - Marks all Vertex parents as runDone
func skipParents(v *Vertex) {
	Logger.Printf("skip parents for %s\n", v.ID)
	for _, c := range v.Parents {
		c.status = runSkip
		skipParents(c)
	}
}

// DephFirstSort - Returns a sorted list with the Vertices
// https://en.wikipedia.org/wiki/Topological_sorting#Depth-first_search
// It returns ErrorGraphHasCycle is there are cycles.
func (g *Graph) DephFirstSort() ([]*Vertex, error) {
	var err error

	sorted := []*Vertex{}
	status := map[ID]visitStatus{}

	for _, vertex := range g.Vertices {
		status[vertex.ID] = unvisited
	}

	for _, v := range g.Vertices {
		if status[v.ID] != unvisited {
			continue
		}
		err = visit(&sorted, &status, v)
		if err != nil {
			return sorted, err
		}
	}
	return sorted, nil
}

func visit(sorted *[]*Vertex, status *map[ID]visitStatus, v *Vertex) error {
	if (*status)[v.ID] == traversed {
		return nil
	}
	if (*status)[v.ID] == visited {
		return fmt.Errorf("%w: %s", ErrorGraphHasCycle, v.ID)
	}
	(*status)[v.ID] = visited

	for _, child := range v.Children {
		err := visit(sorted, status, child)
		if err != nil {
			return err
		}
	}
	(*status)[v.ID] = traversed
	*sorted = append(*sorted, v)
	return nil
}
