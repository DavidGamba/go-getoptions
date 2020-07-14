// TODO: Allow defining an Entrypoint into the graph and operate from that point.
// Subgraph concept.
// TODO: Allow for serial operation if desired.
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
		errs []error
	}

	Vertex struct {
		ID       ID
		Task     *Task
		Children []*Vertex
		Parents  []*Vertex
		status   runStatus
	}

	Graph struct {
		TickerDuration time.Duration
		Vertices       map[ID]*Vertex
		dotDiagram     string
		// Track AddTask and TaskDependensOn errors
		errs []error
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
var ErrorTaskDuplicate = fmt.Errorf("graph already contains task definition")
var ErrorTaskNotFound = fmt.Errorf("task not found in graph")
var ErrorTaskDependencyDuplicate = fmt.Errorf("task dependency already defined")
var ErrorGraphHasCycle = fmt.Errorf("Graph has a cycle")
var ErrorAddTaskOrDependency = fmt.Errorf("errors found in AddTask or TaskDependensOn stages")
var ErrorRunTask = fmt.Errorf("errors during run")

// ErrorSkipParents - Allows for conditional tasks that allow a task to Skip all parent tasks without failing the run
var ErrorSkipParents = fmt.Errorf("skip parents without failing")

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
		errs: make([]error, 0),
	}
}

// Add - Adds a new task to the TaskMap
func (tm *TaskMap) Add(id string, fn getoptions.CommandFn) *Task {
	if id == "" {
		tm.errs = append(tm.errs, ErrorTaskID)
	}
	if fn == nil {
		tm.errs = append(tm.errs, fmt.Errorf("%w for %s", ErrorTaskFn, id))
	}
	newTask := NewTask(id, fn)
	tm.m[id] = newTask
	return newTask
}

// Get - Gets the task form the TaskMap, if not found returns an empty Task.
func (tm *TaskMap) Get(id string) *Task {
	if t, ok := tm.m[id]; ok {
		return t
	}
	return NewTask(id, nil)
}

// Validate - Verifies that there are no errors in the TaskMap.
func (tm *TaskMap) Validate() error {
	return nil
}

// NewGraph - Create a graph that allows running Tasks in parallel.
func NewGraph() *Graph {
	return &Graph{
		TickerDuration: 1 * time.Millisecond,
		Vertices:       make(map[ID]*Vertex, 0),
	}
}

// String - Returns a dot diagram of the graph.
func (g *Graph) String() string {
	dotDiagramHeader := `
digraph G {
	rankdir = TB;
`

	dotDiagramFooter := `}`

	return dotDiagramHeader + g.dotDiagram + dotDiagramFooter
}

// CreateTask - Helper to avoid having to call NewTask and then AddTask if there is no need for reusable Tasks.
func (g *Graph) CreateTask(id string, fn getoptions.CommandFn) error {
	if id == "" {
		return ErrorTaskID
	}
	if fn == nil {
		return ErrorTaskFn
	}
	t := &Task{
		ID: ID(id),
		Fn: fn,
		sm: sync.Mutex{},
	}
	return g.AddTask(t)
}

// AddTask - Add Task to graph.
func (g *Graph) AddTask(t *Task) error {
	if t == nil {
		g.errs = append(g.errs, ErrorTaskNil)
		return ErrorTaskNil
	}
	if t.ID == "" {
		g.errs = append(g.errs, ErrorTaskID)
		return ErrorTaskID
	}
	if t.Fn == nil {
		err := fmt.Errorf("%w for %s", ErrorTaskFn, t.ID)
		g.errs = append(g.errs, err)
		return err
	}
	if _, ok := g.Vertices[t.ID]; ok {
		err := fmt.Errorf("%w: %s", ErrorTaskDuplicate, t.ID)
		g.errs = append(g.errs, err)
		return err
	}
	g.dotDiagram += fmt.Sprintf("\t\"%s\";\n", t.ID)
	g.Vertices[t.ID] = &Vertex{
		ID:       t.ID,
		Task:     t,
		Children: make([]*Vertex, 0),
		Parents:  make([]*Vertex, 0),
		status:   runPending,
	}
	return nil
}

// TaskDependensOn - Allows defining the edges of the Task graph.
func (g *Graph) TaskDependensOn(t string, tDependencies ...string) error {
	task, ok := g.Vertices[ID(t)]
	if !ok {
		err := fmt.Errorf("%w: %s", ErrorTaskNotFound, t)
		g.errs = append(g.errs, err)
		return err
	}
	for _, tDependency := range tDependencies {
		dependency, ok := g.Vertices[ID(tDependency)]
		if !ok {
			err := fmt.Errorf("%w: %s", ErrorTaskNotFound, t)
			g.errs = append(g.errs, err)
			return err
		}
		for _, c := range task.Children {
			if c.ID == dependency.ID {
				err := fmt.Errorf("%w: %s -> %s", ErrorTaskDependencyDuplicate, task.ID, dependency.ID)
				g.errs = append(g.errs, err)
				return err
			}
		}
		g.dotDiagram += fmt.Sprintf("\t\"%s\" -> \"%s\";\n", t, tDependency)
		task.Children = append(task.Children, dependency)
		dependency.Parents = append(dependency.Parents, task)
	}
	return nil
}

// Run - Execute the graph tasks in parallel where possible.
// It checks for tasks updates every 1 Millisecond by default.
// Modify using the graph.TickerDuration
func (g *Graph) Run(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	runStart := time.Now()

	if len(g.errs) != 0 {
		msg := ""
		for _, e := range g.errs {
			msg += fmt.Sprintf("> %s\n", e)
		}
		return fmt.Errorf("%w:\n%s", ErrorAddTaskOrDependency, msg)
	}

	if len(g.Vertices) == 0 {
		return nil
	}

	// Check for cycles
	_, err := g.DephFirstSort()
	if err != nil {
		return err
	}

	TaskErrors := []error{}
	type IDErr struct {
		ID    ID
		Error error
	}
	done := make(chan IDErr)
	count := 0
LOOP:
	for {
		select {
		case iderr := <-done:
			count++
			g.Vertices[iderr.ID].status = runDone
			if count >= len(g.Vertices) {
				break LOOP
			}
			if iderr.Error != nil {
				Logger.Printf("Task %s error: %s\n", iderr.ID, iderr.Error)
				if !errors.Is(iderr.Error, ErrorSkipParents) {
					TaskErrors = append(TaskErrors, iderr.Error)
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
			if !ok {
				time.Sleep(g.TickerDuration)
				// TODO: Add a timeout to not wait infinitely for a task.
				continue
			}
			if v.status == runSkip {
				Logger.Printf("Skipped Task %s\n", v.ID)
				v.status = runDone
				go func(done chan IDErr, v *Vertex) {
					done <- IDErr{v.ID, nil}
				}(done, v)
				continue
			}
			if len(TaskErrors) > 0 {
				Logger.Printf("Skipped Task %s\n", v.ID)
				v.status = runDone
				go func(done chan IDErr, v *Vertex) {
					done <- IDErr{v.ID, fmt.Errorf("skipped %s", v.ID)}
				}(done, v)
				continue
			}
			v.status = runInProgress
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
	if len(TaskErrors) > 0 {
		msg := ""
		for _, e := range TaskErrors {
			msg += fmt.Sprintf("> %s\n", e)
		}
		return fmt.Errorf("%w:\n%s", ErrorRunTask, msg)
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
func (g *Graph) getNextVertex() (*Vertex, bool, bool) {
	doneCount := 0
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
