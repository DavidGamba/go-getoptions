package dag

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

func setupLoggingWithoutTime() *bytes.Buffer {
	s := ""
	buf := bytes.NewBufferString(s)
	Logger.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	Logger.SetOutput(buf)
	return buf
}

func TestColor(t *testing.T) {
	buf := setupLoggingWithoutTime()
	t.Cleanup(func() { t.Log(buf.String()) })

	var err error

	sm := sync.Mutex{}
	results := []int{}
	generateFn := func(n int) getoptions.CommandFn {
		return func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
			time.Sleep(30 * time.Millisecond)
			if n == 2 {
				return fmt.Errorf("failure reason")
			}
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

	g := NewGraph("test graph").SetSerial()
	g.UseColor = true
	g.TaskDependsOn(tm.Get("t1"), tm.Get("t2"), tm.Get("t3"))
	g.TaskDependsOn(tm.Get("t2"), tm.Get("t3"))

	// Validate before running
	err = g.Validate(tm)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}

	err = g.Run(context.Background(), nil, nil)
	var errs *Errors
	if err == nil || !errors.As(err, &errs) {
		t.Fatalf("Unexpected error: %s\n", err)
	}
	if len(errs.Errors) != 2 {
		t.Fatalf("Unexpected error size, %d: %s\n", len(errs.Errors), err)
	}
	if errs.Errors[0].Error() != "Task test graph:t2 error: failure reason" {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[0])
	}
	if !errors.Is(errs.Errors[1], ErrorTaskSkipped) {
		t.Fatalf("Unexpected error: %s\n", errs.Errors[1])
	}
	if len(results) != 1 || results[0] != 3 {
		t.Errorf("Wrong list: %v, len: %d, 0: %d\n", results, len(results), results[0])
	}

	expected := "\033[34mRunning Task \033[0m\033[36;1mtest graph:t3\033[0m\n" +
		"\033[34mCompleted Task \033[0m\033[36;1mtest graph:t3\033[0m\033[34m in 00m:00s\033[0m\n" +
		"\033[34mRunning Task \033[0m\033[36;1mtest graph:t2\033[0m\n" +
		"\033[34mCompleted Task \033[0m\033[36;1mtest graph:t2\033[0m\033[34m in 00m:00s\033[0m\n" +
		"\033[31mTask \033[0m\033[35;1mtest graph:t2\033[0m\033[31m error: failure reason\033[0m\n" +
		"\033[31mTask \033[0m\033[35;1mtest graph:t1\033[0m\033[31m error: skipped\033[0m\n" +
		"\033[34mCompleted \033[0m\033[36;1mtest graph\033[0m\033[34m Run in 00m:00s\033[0m\n"
	if buf.String() != expected {
		t.Errorf("Wrong output:\n'%s'\nexpected:\n'%s'\n", buf.String(), expected)
	}
}
