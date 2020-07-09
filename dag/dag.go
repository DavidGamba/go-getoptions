package dag

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

var Logger = log.New(os.Stderr, "", log.LstdFlags)

type (
	ID string

	Task struct {
		ID ID
		Fn getoptions.CommandFn
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
	}

	runStatus int

	visitStatus int
)

const (
	runPending runStatus = iota
	runInProgress
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

var ErrorTaskDuplicate = fmt.Errorf("graph already contains task definition")
var ErrorTaskNotFound = fmt.Errorf("task not found in graph")
var ErrorTaskDependencyDuplicate = fmt.Errorf("task dependency already defined")
var ErrorGraphHasCycle = fmt.Errorf("Graph has a cycle")

// NewTask - Allows creating a reusable Task that can be passed to multiple graphs.
func NewTask(id string, fn getoptions.CommandFn) *Task {
	return &Task{
		ID: ID(id),
		Fn: fn,
	}
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
	t := &Task{
		ID: ID(id),
		Fn: fn,
	}
	return g.AddTask(t)
}

// AddTask - Add Task to graph.
func (g *Graph) AddTask(t *Task) error {
	if _, ok := g.Vertices[t.ID]; ok {
		return fmt.Errorf("%w: %s", ErrorTaskDuplicate, t.ID)
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

// TaskDependensOn - Allows defining the edges of the Task graph.
func (g *Graph) TaskDependensOn(t ID, tDependencies ...ID) error {
	task, ok := g.Vertices[t]
	if !ok {
		return fmt.Errorf("%w: %s", ErrorTaskNotFound, t)
	}
	for _, tDependency := range tDependencies {
		dependency, ok := g.Vertices[tDependency]
		if !ok {
			return fmt.Errorf("%w: %s", ErrorTaskNotFound, t)
		}
		for _, c := range task.Children {
			if c.ID == dependency.ID {
				return fmt.Errorf("%w: %s -> %s", ErrorTaskDependencyDuplicate, task.ID, dependency.ID)
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
	// Check for cycles
	_, err := g.DephFirstSort()
	if err != nil {
		return err
	}

	done := make(chan ID)
	count := 0
LOOP:
	for {
		select {
		case id := <-done:
			count++
			Logger.Printf("count %d\n", count)
			g.Vertices[id].status = runDone
			if count >= len(g.Vertices) {
				break LOOP
			}
		default:
			v, ok := g.getNextVertex()
			if !ok {
				time.Sleep(g.TickerDuration)
				continue
			}
			v.status = runInProgress
			Logger.Printf("Running Task %s\n", v.ID)
			go func(done chan ID, v *Vertex) {
				start := time.Now()
				v.Task.Fn(ctx, opt, args)
				Logger.Printf("Completed Task %s in %s\n", v.ID, durationStr(time.Since(start)))
				done <- v.ID
			}(done, v)
		}
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
func (g *Graph) getNextVertex() (*Vertex, bool) {
	for _, vertex := range g.Vertices {
		if vertex.status != runPending {
			continue
		}
		if len(vertex.Children) == 0 {
			return vertex, true
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
			return vertex, true
		}
	}
	return nil, false
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
	fmt.Printf("sorted: %v\n", sorted)
	for _, e := range sorted {
		fmt.Printf(" %s ", e.ID)
	}
	fmt.Println()
	return sorted, nil
}

func visit(sorted *[]*Vertex, status *map[ID]visitStatus, v *Vertex) error {
	Logger.Printf("visiting %s\n", v.ID)
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
