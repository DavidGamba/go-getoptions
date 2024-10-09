// This file is part of go-getoptions.
//
// Copyright (C) 2015-2024  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// TODO: Allow defining an Entry point into the graph and operate from that point.
// Subgraph concept.
//
// TODO: Aggregate logs into a buffer and print the logs when the task completes so they aren't merged together when running in parallel?
//   This one is not possible since Go's stdout and stderr are a single global variable and we can't redirect it per Fn call unless the function was running in some sort of sandbox which it is not.
//
// TODO: Pass context to subtask.

/*
Package dag - Lightweight Directed Acyclic Graph (DAG) Build System.

It allows building a list of tasks and then running the tasks in different DAG trees.

The tree dependencies are calculated and tasks that have met their dependencies are run in parallel.
Max parallelism can be set and there is an option to run it serially for cases where user interaction is required.
*/
package dag

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

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
		Retries  int
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
		maxParallel    int
		bufferOutput   bool
		bufferWriter   io.Writer
		bufferMutex    sync.Mutex
		UseColor       bool
		InfoColor      string
		InfoBoldColor  string
		ErrorColor     string
		ErrorBoldColor string
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

var (
	ErrorTaskNil                 = fmt.Errorf("nil task given")
	ErrorTaskID                  = fmt.Errorf("missing task ID")
	ErrorTaskFn                  = fmt.Errorf("missing task function")
	ErrorTaskDuplicate           = fmt.Errorf("task definition already exists")
	ErrorTaskNotFound            = fmt.Errorf("task not found")
	ErrorTaskDependencyDuplicate = fmt.Errorf("task dependency already defined")
	ErrorGraphHasCycle           = fmt.Errorf("graph has a cycle")
	ErrorTaskSkipped             = fmt.Errorf("skipped")
)

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
		maxParallel:    1_000_000,
		InfoColor:      "34",
		InfoBoldColor:  "36;1",
		ErrorColor:     "31",
		ErrorBoldColor: "35;1",
	}
}

// SetSerial - The call to Run() will run the Tasks serially.
// Useful when the tasks require user input and the user needs to see logs in order to make a decision.
func (g *Graph) SetSerial() *Graph {
	g.serial = true
	return g
}

// SetMaxParallel - Limit concurrency.
func (g *Graph) SetMaxParallel(max int) *Graph {
	if max > 0 {
		g.maxParallel = max
	}
	return g
}

// SetOutputBuffer - Adds a buffer in the context passed to the task that allows the task logging to be sent to that buffer.
// At the end of the task, the in memory buffered output will be written to the given io.Writer.
//
// The context keys are 'StdoutBuffer' and 'StderrBuffer' which can be retrieved with the helper functions.
// The helper functions default to os.Stdout and os.Stderr when no buffering is defined.
//
//	g.SetOutputBuffer(os.Stdout)
//	stdout := dag.Stdout(ctx)
//	stderr := dag.Stderr(ctx)
//	fmt.Fprintf(stdout, "Output")
//	fmt.Fprintf(stderr, "Error")
//
// NOTE: Even though both stdout and stderr contex keys are provided, currently both are combined into a single output and written into the given io.Writer.
func (g *Graph) SetOutputBuffer(w io.Writer) *Graph {
	g.bufferOutput = true
	g.bufferWriter = w
	g.bufferMutex = sync.Mutex{}
	return g
}

type ContextKey string

func Stdout(ctx context.Context) io.Writer {
	if v := ctx.Value(ContextKey("StdoutBuffer")); v != nil {
		if writer, ok := v.(io.Writer); ok {
			return writer
		}
	}
	return os.Stdout
}

func Stderr(ctx context.Context) io.Writer {
	if v := ctx.Value(ContextKey("StderrBuffer")); v != nil {
		if writer, ok := v.(io.Writer); ok {
			return writer
		}
	}
	return os.Stderr
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

// TaskDependsOn - Allows adding tasks to the graph and defining their edges.
func (g *Graph) TaskDependsOn(t *Task, tDependencies ...*Task) {
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

// TaskRetries - Set a number of retries for a task.
func (g *Graph) TaskRetries(t *Task, retries int) {
	vertex, err := g.retrieveOrAddVertex(t)
	if err != nil {
		g.errs.Errors = append(g.errs.Errors, err)
		return
	}
	vertex.Retries = retries
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
	_, err := g.DepthFirstSort()
	if err != nil {
		return err
	}

	handledContext := false
	type IDErr struct {
		ID    ID
		Error error
	}
	done := make(chan IDErr)
	semaphore := make(chan struct{}, g.maxParallel)
LOOP:
	for {
		select {
		case iderr := <-done:
			g.Vertices[iderr.ID].status = runDone
			if iderr.Error != nil {
				err := fmt.Errorf("Task %s:%s error: %w", g.Name, iderr.ID, iderr.Error)
				Logger.Printf(g.colorError("Task ")+g.colorErrorBold("%s:%s")+g.colorError(" error: %s\n"), g.Name, iderr.ID, iderr.Error)
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
				Logger.Print(g.colorError("Cancellation received or time out reached, allowing in-progress tasks to finish, skipping the rest.\n"))
				g.errs.Errors = append(g.errs.Errors, fmt.Errorf("cancellation received or time out reached"))
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
				Logger.Printf(g.colorError("Skipped Task ")+g.colorErrorBold("%s:%s\n"), g.Name, v.ID)
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
			go func(ctx context.Context, done chan IDErr, v *Vertex) {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				Logger.Printf(g.colorInfo("Running Task ")+g.colorInfoBold("%s:%s\n"), g.Name, v.ID)
				start := time.Now()
				v.Task.Lock()
				defer v.Task.Unlock()
				combinedBuffer := bytes.Buffer{}
				// TODO: It would be great to be able to color the output independently here
				stdoutBuffer := &combinedBuffer
				stderrBuffer := &combinedBuffer
				if g.bufferOutput {
					ctx = context.WithValue(ctx, ContextKey("StdoutBuffer"), stdoutBuffer)
					ctx = context.WithValue(ctx, ContextKey("StderrBuffer"), stderrBuffer)
				}
				var err error
				for i := 0; i <= v.Retries; i++ {
					err = v.Task.Fn(ctx, opt, args)
					if g.bufferOutput {
						g.bufferMutex.Lock()
						_, _ = combinedBuffer.WriteTo(g.bufferWriter)
						g.bufferMutex.Unlock()
					}
					Logger.Printf(g.colorInfo("Completed Task ")+g.colorInfoBold("%s:%s")+g.colorInfo(" in %s\n"), g.Name, v.ID, durationStr(time.Since(start)))
					if err == nil {
						break
					}
					if err != nil && i < v.Retries {
						Logger.Printf(g.colorError("Task ")+g.colorErrorBold("%s:%s")+g.colorError(" error: %s"), g.Name, v.ID, err)
						Logger.Printf(g.colorInfo("Retrying (%d/%d) Task %s:%s\n"), i+1, v.Retries, g.Name, v.ID)
					}
				}
				done <- IDErr{v.ID, err}
			}(ctx, done, v)
		}
	}
	Logger.Printf(g.colorInfo("Completed ")+g.colorInfoBold("%s")+g.colorInfo(" Run in %s\n"), g.Name, durationStr(time.Since(runStart)))

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
	// Logger.Printf("skip parents for %s\n", v.ID)
	for _, c := range v.Parents {
		c.status = runSkip
		skipParents(c)
	}
}

// DepthFirstSort - Returns a sorted list with the Vertices
// https://en.wikipedia.org/wiki/Topological_sorting#Depth-first_search
// It returns ErrorGraphHasCycle is there are cycles.
func (g *Graph) DepthFirstSort() ([]*Vertex, error) {
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
