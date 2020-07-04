package build

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sync"

	"github.com/DavidGamba/go-getoptions"
)

var Debug = log.New(os.Stderr, "", log.LstdFlags)

// mutexMap - Implements a shared mutex that multiple callers can call concurrently.
// The mutex is backed by a buffered channel
//
// Example:
//     var sharedMutex = &build.NewMutexMap()
//
//     done := sharedMutex.Lock("key")
//     defer done()
type mutexMap struct {
	sm sync.Map
}

func NewMutexMap() *mutexMap {
	return &mutexMap{
		sm: sync.Map{},
	}
}

func (m *mutexMap) Lock(name string) func() {
	_, ok := m.sm.Load(name)
	if !ok {
		c := make(chan struct{}, 1)
		m.sm.Store(name, c)
	}
	v, ok := m.sm.Load(name)
	if ok {
		v.(chan struct{}) <- struct{}{}
	}
	return func() {
		<-v.(chan struct{})
	}
}

// LockTask - Given an identifier for the closure, it ensures that only one version of the closure with the given ID is running at a time.
func (m *mutexMap) LockTask(id string) func() {
	pc := make([]uintptr, 15)
	n := runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc[:n])
	frame, _ := frames.Next()
	fnName := frame.Function
	done := m.Lock(fmt.Sprintf("%s-%s", fnName, id))
	Debug.Printf("Locking Task: %s\n", fmt.Sprintf("%s-%s", fnName, id))
	return func() {
		done()
		Debug.Printf("Unlocking Task: %s\n", fmt.Sprintf("%s-%s", fnName, id))
	}
}

func DependsOnSerial(ctx context.Context, opt *getoptions.GetOpt, args []string, fns ...getoptions.CommandFn) error {
	for _, fn := range fns {
		err := fn(ctx, opt, args)
		if err != nil {
			return fmt.Errorf("dependency errors: %w", err)
		}
	}
	return nil
}

// TODO: Error code with error wrap and ErrorCodeN(1) for example.

func DependsOn(m *mutexMap, ctx context.Context, opt *getoptions.GetOpt, args []string, fns ...getoptions.CommandFn) error {
	wg := sync.WaitGroup{}
	errs := make([]error, len(fns))
	for i, fn := range fns {
		wg.Add(1)
		go func(i int, wg *sync.WaitGroup, fn getoptions.CommandFn) {
			defer wg.Done()
			err := RunTask(m, fn, ctx, opt, args)
			if err != nil {
				errs[i] = err
			}
		}(i, &wg, fn)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return fmt.Errorf("dependency errors: %w", err)
		}
	}
	return nil
}

// RunTask - Allows to run a task inside a mutex lock so that in can only run one at a time.
// The task is expected to be idempotent.
// The mutex is not applied to closures, for closures use TaskMutex.
func RunTask(m *mutexMap, fn getoptions.CommandFn, ctx context.Context, opt *getoptions.GetOpt, args []string) error {
	fnName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	Debug.Printf("Running Task: %s\n", fnName)
	defer Debug.Printf("Completed Task: %s\n", fnName)
	re := regexp.MustCompile(`\.func\d+$`)
	if !re.MatchString(fnName) {
		Debug.Printf("Locking Task: %s\n", fnName)
		done := m.Lock(fnName)
		defer func() {
			Debug.Printf("Unlocking Task: %s\n", fnName)
			done()
		}()
	}
	return fn(ctx, opt, args)
}
