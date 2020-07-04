package dag

import (
	"fmt"

	"github.com/DavidGamba/go-getoptions"
)

type ID string

type Task struct {
	ID ID
	Fn getoptions.CommandFn
}

func NewTask(id ID, fn getoptions.CommandFn) *Task {
	return &Task{
		ID: id,
		Fn: fn,
	}
}

type Vertex struct {
	ID       ID
	Task     *Task
	Children []*Vertex
}

type Graph struct {
	Vertices map[ID]*Vertex
}

var ErrorTaskDuplicate = fmt.Errorf("graph already contains task definition")
var ErrorTaskNotFound = fmt.Errorf("task not found in graph")
var ErrorTaskDependencyDuplicate = fmt.Errorf("task dependency already defined")
var ErrorGraphHasCycle = fmt.Errorf("Graph has a cycle")

func NewGraph() *Graph {
	return &Graph{make(map[ID]*Vertex, 0)}
}

func (g *Graph) AddTask(t *Task) error {
	if _, ok := g.Vertices[t.ID]; ok {
		return fmt.Errorf("%w: %s", ErrorTaskDuplicate, t.ID)
	}
	g.Vertices[t.ID] = &Vertex{t.ID, t, make([]*Vertex, 0)}
	return nil
}

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
		task.Children = append(task.Children, dependency)
	}
	return nil
}
