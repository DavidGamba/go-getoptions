package build

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/DavidGamba/go-getoptions"
)

func TestMutexMap(t *testing.T) {
	m := NewMutexMap()
	count := 0
	countMutex := sync.Mutex{}
	fn := func(m *mutexMap, wg *sync.WaitGroup) {
		defer wg.Done()
		done := m.Lock("fn")
		defer done()
		time.Sleep(2 * time.Millisecond)
		countMutex.Lock()
		count++
		countMutex.Unlock()
	}
	before := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go fn(m, &wg)
	}
	wg.Wait()
	if time.Since(before) < 10*time.Millisecond {
		t.Errorf("Wrong timing %v\n", time.Since(before))
	}
}

func TestDependsOnSerial(t *testing.T) {
	results := []int{}
	fn1 := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		results = append(results, 1)
		return nil
	}
	fn2 := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		results = append(results, 2)
		return nil
	}
	fn3 := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		results = append(results, 3)
		return nil
	}
	err := DependsOnSerial(nil, nil, nil, fn1, fn2, fn3)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
	if !reflect.DeepEqual(results, []int{1, 2, 3}) {
		t.Errorf("Got %v, Want %v", results, []int{1, 2, 3})
	}
}

func TestDependsOn(t *testing.T) {
	m := NewMutexMap()
	results := []int{}

	sm := sync.Mutex{}
	fn1 := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		done := m.LockTask("fn1")
		defer done()
		sm.Lock()
		results = append(results, 1)
		sm.Unlock()
		return nil
	}
	fn2 := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		done := m.LockTask("fn2")
		defer done()
		DependsOn(m, ctx, opt, args, fn1)
		sm.Lock()
		results = append(results, 2)
		sm.Unlock()
		return nil
	}
	fn3 := func(ctx context.Context, opt *getoptions.GetOpt, args []string) error {
		done := m.LockTask("fn3")
		defer done()
		DependsOn(m, ctx, opt, args, fn1, fn2)
		sm.Lock()
		results = append(results, 3)
		sm.Unlock()
		return nil
	}
	err := DependsOn(m, nil, nil, nil, fn3)
	if err != nil {
		t.Errorf("Unexpected error: %s\n", err)
	}
	if !reflect.DeepEqual(results, []int{1, 1, 2, 3}) && !reflect.DeepEqual(results, []int{1, 2, 1, 3}) {
		t.Errorf("Got %v, Want %v or %v", results, []int{1, 1, 2, 3}, []int{1, 2, 1, 3})
	}
}
