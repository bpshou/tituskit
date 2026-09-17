package deploader

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	t.Run("no modules", func(t *testing.T) {
		Load([]Module{})
	})

	t.Run("single module no deps", func(t *testing.T) {
		var called bool
		Load([]Module{
			{Name: "a", Fn: func() { called = true }},
		})
		if !called {
			t.Fatal("module a was not called")
		}
	})

	t.Run("multiple modules no deps run concurrently", func(t *testing.T) {
		var mu sync.Mutex
		order := make([]string, 0)
		Load([]Module{
			{Name: "a", Fn: func() {
				mu.Lock()
				order = append(order, "a")
				mu.Unlock()
			}},
			{Name: "b", Fn: func() {
				mu.Lock()
				order = append(order, "b")
				mu.Unlock()
			}},
			{Name: "c", Fn: func() {
				mu.Lock()
				order = append(order, "c")
				mu.Unlock()
			}},
		})
		if len(order) != 3 {
			t.Fatalf("expected 3 modules called, got %d", len(order))
		}
	})

	t.Run("dependency ordering respected", func(t *testing.T) {
		var mu sync.Mutex
		ran := make(map[string]bool)
		Load([]Module{
			{Name: "a", Fn: func() {
				mu.Lock()
				ran["a"] = true
				mu.Unlock()
			}},
			{Name: "b", Fn: func() {
				mu.Lock()
				if !ran["a"] {
					t.Error("b ran before a")
				}
				ran["b"] = true
				mu.Unlock()
			}, Deps: []string{"a"}},
			{Name: "c", Fn: func() {
				mu.Lock()
				if !ran["a"] || !ran["b"] {
					t.Error("c ran before a or b")
				}
				ran["c"] = true
				mu.Unlock()
			}, Deps: []string{"a", "b"}},
		})
	})

	t.Run("diamond dependency", func(t *testing.T) {
		var mu sync.Mutex
		ran := make(map[string]bool)
		Load([]Module{
			{Name: "a", Fn: func() {
				mu.Lock()
				ran["a"] = true
				mu.Unlock()
			}},
			{Name: "b", Fn: func() {
				mu.Lock()
				if !ran["a"] {
					t.Error("b ran before a")
				}
				ran["b"] = true
				mu.Unlock()
			}, Deps: []string{"a"}},
			{Name: "c", Fn: func() {
				mu.Lock()
				if !ran["a"] {
					t.Error("c ran before a")
				}
				ran["c"] = true
				mu.Unlock()
			}, Deps: []string{"a"}},
			{Name: "d", Fn: func() {
				mu.Lock()
				if !ran["b"] || !ran["c"] {
					t.Error("d ran before b or c")
				}
				ran["d"] = true
				mu.Unlock()
			}, Deps: []string{"b", "c"}},
		})
	})

	t.Run("unknown dependency panics", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for unknown dependency, got none")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic, got %T: %v", r, r)
			}
			if msg != `deploader: module "b" depends on unknown module "nonexistent"` {
				t.Fatalf("unexpected panic message: %s", msg)
			}
		}()
		Load([]Module{
			{Name: "a", Fn: func() {}},
			{Name: "b", Fn: func() {}, Deps: []string{"nonexistent"}},
		})
	})

	t.Run("duplicate module name panics", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for duplicate name, got none")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic, got %T: %v", r, r)
			}
			if msg != "deploader: duplicate module name: a" {
				t.Fatalf("unexpected panic message: %s", msg)
			}
		}()
		Load([]Module{
			{Name: "a", Fn: func() {}},
			{Name: "a", Fn: func() {}},
		})
	})

	t.Run("all modules called exactly once", func(t *testing.T) {
		var mu sync.Mutex
		counts := make(map[string]int)
		Load([]Module{
			{Name: "a", Fn: func() {
				mu.Lock()
				counts["a"]++
				mu.Unlock()
			}},
			{Name: "b", Fn: func() {
				mu.Lock()
				counts["b"]++
				mu.Unlock()
			}, Deps: []string{"a"}},
			{Name: "c", Fn: func() {
				mu.Lock()
				counts["c"]++
				mu.Unlock()
			}, Deps: []string{"a", "b"}},
		})
		for _, name := range []string{"a", "b", "c"} {
			if counts[name] != 1 {
				t.Fatalf("module %s called %d times, expected 1", name, counts[name])
			}
		}
	})

	t.Run("concurrent safety with timing", func(t *testing.T) {
		var mu sync.Mutex
		var order []string
		Load([]Module{
			{Name: "fast", Fn: func() {
				time.Sleep(10 * time.Millisecond)
				mu.Lock()
				order = append(order, "fast")
				mu.Unlock()
			}},
			{Name: "slow", Fn: func() {
				time.Sleep(50 * time.Millisecond)
				mu.Lock()
				order = append(order, "slow")
				mu.Unlock()
			}},
			{Name: "after_slow", Fn: func() {
				mu.Lock()
				order = append(order, "after_slow")
				mu.Unlock()
			}, Deps: []string{"slow"}},
		})
		if order[len(order)-1] != "after_slow" {
			t.Fatalf("after_slow should be last, got order: %v", order)
		}
	})
}

func TestLoad_NilFunction(t *testing.T) {
	if os.Getenv("TEST_NIL_FN") == "1" {
		Load([]Module{
			{Name: "nil_fn", Fn: nil},
		})
		return
	}

	t.Run("nil function causes fatal crash", func(t *testing.T) {
		cmd := exec.Command(os.Args[0], "-test.run=TestLoad_NilFunction")
		cmd.Env = append(os.Environ(), "TEST_NIL_FN=1")
		err := cmd.Run()
		if err == nil {
			t.Fatal("expected process to exit with error for nil function")
		}
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatalf("expected exit error, got %T: %v", err, err)
		}
	})
}

func TestLoad_SelfDependency(t *testing.T) {
	// 自依赖不会被校验拦截，会在 goroutine 内等待自身导致永久阻塞（死锁）
	done := make(chan struct{})
	go func() {
		defer close(done)
		Load([]Module{
			{Name: "a", Fn: func() {}, Deps: []string{"a"}},
		})
	}()

	select {
	case <-done:
		t.Fatal("expected deadlock for self dependency, but Load returned")
	case <-time.After(100 * time.Millisecond):
		// 预期：goroutine 阻塞等待自身
	}
}

func TestLoad_CyclicDependency(t *testing.T) {
	t.Run("two-node cycle", func(t *testing.T) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			Load([]Module{
				{Name: "a", Fn: func() {}, Deps: []string{"b"}},
				{Name: "b", Fn: func() {}, Deps: []string{"a"}},
			})
		}()

		select {
		case <-done:
			t.Fatal("expected deadlock for cyclic dependency, but Load returned")
		case <-time.After(100 * time.Millisecond):
			// expected: goroutines blocked waiting on each other
		}
	})

	t.Run("three-node cycle", func(t *testing.T) {
		done := make(chan struct{})
		go func() {
			defer close(done)
			Load([]Module{
				{Name: "a", Fn: func() {}, Deps: []string{"b"}},
				{Name: "b", Fn: func() {}, Deps: []string{"c"}},
				{Name: "c", Fn: func() {}, Deps: []string{"a"}},
			})
		}()

		select {
		case <-done:
			t.Fatal("expected deadlock for cyclic dependency, but Load returned")
		case <-time.After(100 * time.Millisecond):
			// expected
		}
	})
}

func TestLoad_EmptyModuleName(t *testing.T) {
	t.Run("single module with empty name", func(t *testing.T) {
		var called bool
		Load([]Module{
			{Name: "", Fn: func() { called = true }},
		})
		if !called {
			t.Fatal("module with empty name was not called")
		}
	})

	t.Run("duplicate empty names panics", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for duplicate empty name")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic, got %T: %v", r, r)
			}
			if msg != "deploader: duplicate module name: " {
				t.Fatalf("unexpected panic message: %s", msg)
			}
		}()
		Load([]Module{
			{Name: "", Fn: func() {}},
			{Name: "", Fn: func() {}},
		})
	})
}

func TestLoad_FnPanics(t *testing.T) {
	if os.Getenv("TEST_FN_PANIC") != "" {
		switch os.Getenv("TEST_FN_PANIC") {
		case "propagate":
			Load([]Module{
				{Name: "a", Fn: func() { panic("module a failed") }},
			})
		case "dependency":
			Load([]Module{
				{Name: "a", Fn: func() { panic("a failed") }},
				{Name: "b", Fn: func() {}, Deps: []string{"a"}},
			})
		case "unrelated":
			Load([]Module{
				{Name: "a", Fn: func() { panic("a failed") }},
				{Name: "b", Fn: func() {}},
				{Name: "c", Fn: func() {}, Deps: []string{"b"}},
			})
		}
		return
	}

	t.Run("panic in module function crashes process", func(t *testing.T) {
		err := runPanicScenario(t, "propagate")
		if err == nil {
			t.Fatal("expected crash for panic in module function")
		}
	})

	t.Run("panic in dependency crashes process", func(t *testing.T) {
		err := runPanicScenario(t, "dependency")
		if err == nil {
			t.Fatal("expected crash for panic in dependency")
		}
	})

	t.Run("panic in unrelated module still crashes process", func(t *testing.T) {
		err := runPanicScenario(t, "unrelated")
		if err == nil {
			t.Fatal("expected crash for panic in unrelated module")
		}
	})
}

func runPanicScenario(t *testing.T, scenario string) error {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=TestLoad_FnPanics")
	cmd.Env = append(os.Environ(), "TEST_FN_PANIC="+scenario)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	if _, ok := err.(*exec.ExitError); !ok {
		t.Fatalf("expected exit error, got %T: %v", err, err)
	}
	return err
}

func TestLoad_NilDepsSlice(t *testing.T) {
	var called bool
	Load([]Module{
		{Name: "a", Fn: func() { called = true }, Deps: nil},
	})
	if !called {
		t.Fatal("module with nil Deps was not called")
	}
}

func TestLoad_EmptyDepsSlice(t *testing.T) {
	var called bool
	Load([]Module{
		{Name: "a", Fn: func() { called = true }, Deps: []string{}},
	})
	if !called {
		t.Fatal("module with empty Deps was not called")
	}
}

func TestLoad_FanOutDependency(t *testing.T) {
	var mu sync.Mutex
	ran := make(map[string]bool)
	const n = 20

	modules := make([]Module, 0, n+1)
	modules = append(modules, Module{
		Name: "root",
		Fn: func() {
			mu.Lock()
			ran["root"] = true
			mu.Unlock()
		},
	})

	for i := 0; i < n; i++ {
		name := "child_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		modules = append(modules, Module{
			Name: name,
			Fn: func() {
				mu.Lock()
				ran[name] = true
				mu.Unlock()
			},
			Deps: []string{"root"},
		})
	}

	Load(modules)

	mu.Lock()
	defer mu.Unlock()
	if !ran["root"] {
		t.Fatal("root module was not called")
	}
	for i := 0; i < n; i++ {
		name := "child_" + string(rune('a'+i%26)) + string(rune('0'+i/26))
		if !ran[name] {
			t.Fatalf("module %s was not called", name)
		}
	}
}

func TestLoad_DeepChain(t *testing.T) {
	const depth = 50
	var mu sync.Mutex
	ran := make(map[string]bool)

	modules := make([]Module, depth)
	for i := 0; i < depth; i++ {
		name := string(rune('a' + i%26))
		if i >= 26 {
			name += string(rune('0' + i/26))
		}
		modules[i] = Module{
			Name: name,
			Fn: func() {
				mu.Lock()
				ran[name] = true
				mu.Unlock()
			},
		}
		if i > 0 {
			prevName := string(rune('a' + (i-1)%26))
			if i-1 >= 26 {
				prevName += string(rune('0' + (i-1)/26))
			}
			modules[i].Deps = []string{prevName}
		}
	}

	Load(modules)

	mu.Lock()
	defer mu.Unlock()
	if len(ran) != depth {
		t.Fatalf("expected %d modules called, got %d", depth, len(ran))
	}
}

func TestLoad_ComplexGraph(t *testing.T) {
	var mu sync.Mutex
	ran := make(map[string]bool)

	// a -> b, c
	// b -> d
	// c -> d
	// d -> e
	Load([]Module{
		{Name: "a", Fn: func() {
			mu.Lock()
			ran["a"] = true
			mu.Unlock()
		}},
		{Name: "b", Fn: func() {
			mu.Lock()
			if !ran["a"] {
				t.Error("b ran before a")
			}
			ran["b"] = true
			mu.Unlock()
		}, Deps: []string{"a"}},
		{Name: "c", Fn: func() {
			mu.Lock()
			if !ran["a"] {
				t.Error("c ran before a")
			}
			ran["c"] = true
			mu.Unlock()
		}, Deps: []string{"a"}},
		{Name: "d", Fn: func() {
			mu.Lock()
			if !ran["b"] || !ran["c"] {
				t.Error("d ran before b or c")
			}
			ran["d"] = true
			mu.Unlock()
		}, Deps: []string{"b", "c"}},
		{Name: "e", Fn: func() {
			mu.Lock()
			if !ran["d"] {
				t.Error("e ran before d")
			}
			ran["e"] = true
			mu.Unlock()
		}, Deps: []string{"d"}},
	})
}

func TestLoad_MultipleUnknownDeps(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for unknown dependency")
		}
	}()
	Load([]Module{
		{Name: "a", Fn: func() {}, Deps: []string{"x", "y", "z"}},
	})
}

func TestLoad_DuplicateModuleNameWithDeps(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate name")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if msg != "deploader: duplicate module name: a" {
			t.Fatalf("unexpected panic message: %s", msg)
		}
	}()
	Load([]Module{
		{Name: "a", Fn: func() {}, Deps: []string{"b"}},
		{Name: "b", Fn: func() {}},
		{Name: "a", Fn: func() {}, Deps: []string{"b"}},
	})
}

func TestLoad_PanicRecovery(t *testing.T) {
	t.Run("recover from panic in Load", func(t *testing.T) {
		var recovered bool
		func() {
			defer func() {
				if r := recover(); r != nil {
					recovered = true
				}
			}()
			Load([]Module{
				{Name: "a", Fn: func() {}},
				{Name: "a", Fn: func() {}},
			})
		}()
		if !recovered {
			t.Fatal("panic was not recoverable")
		}
	})

	t.Run("recover from unknown dep panic", func(t *testing.T) {
		var recovered bool
		func() {
			defer func() {
				if r := recover(); r != nil {
					recovered = true
				}
			}()
			Load([]Module{
				{Name: "a", Fn: func() {}, Deps: []string{"missing"}},
			})
		}()
		if !recovered {
			t.Fatal("panic was not recoverable")
		}
	})
}

func TestLoad_OrderConsistency(t *testing.T) {
	const runs = 10
	for i := 0; i < runs; i++ {
		var mu sync.Mutex
		var order []string
		Load([]Module{
			{Name: "a", Fn: func() {
				mu.Lock()
				order = append(order, "a")
				mu.Unlock()
			}},
			{Name: "b", Fn: func() {
				mu.Lock()
				order = append(order, "b")
				mu.Unlock()
			}, Deps: []string{"a"}},
		})
		mu.Lock()
		if len(order) != 2 || order[0] != "a" || order[1] != "b" {
			t.Fatalf("run %d: expected [a b], got %v", i, order)
		}
		mu.Unlock()
	}
}

func TestLoad_LargeModuleCount(t *testing.T) {
	const n = 100
	var count int32
	modules := make([]Module, n)
	for i := 0; i < n; i++ {
		modules[i] = Module{
			Name: fmt.Sprintf("module_%d", i),
			Fn: func() {
				atomic.AddInt32(&count, 1)
			},
		}
	}

	Load(modules)

	if atomic.LoadInt32(&count) != n {
		t.Fatalf("expected %d modules called, got %d", n, count)
	}
}

func TestLoad_ErrorReturningFunction(t *testing.T) {
	var mu sync.Mutex
	errs := make(map[string]error)

	Load([]Module{
		{Name: "a", Fn: func() {
			mu.Lock()
			errs["a"] = errors.New("a error")
			mu.Unlock()
		}},
		{Name: "b", Fn: func() {
			mu.Lock()
			errs["b"] = nil
			mu.Unlock()
		}, Deps: []string{"a"}},
	})

	mu.Lock()
	defer mu.Unlock()
	if errs["a"] == nil {
		t.Fatal("expected error from a")
	}
	if errs["b"] != nil {
		t.Fatal("b should not have error")
	}
}

func TestLoad_ChannelBasedOrderVerification(t *testing.T) {
	ch := make(chan string, 10)

	Load([]Module{
		{Name: "a", Fn: func() { ch <- "a" }},
		{Name: "b", Fn: func() { ch <- "b" }, Deps: []string{"a"}},
		{Name: "c", Fn: func() { ch <- "c" }, Deps: []string{"b"}},
	})
	close(ch)

	var result []string
	for v := range ch {
		result = append(result, v)
	}

	if len(result) != 3 {
		t.Fatalf("expected 3 results, got %d", len(result))
	}
	if result[0] != "a" || result[1] != "b" || result[2] != "c" {
		t.Fatalf("expected [a b c], got %v", result)
	}
}
