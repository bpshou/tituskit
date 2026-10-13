package deploader

import (
	"fmt"
	"strings"
	"sync"
)

// Module 定义一个可加载的依赖模块
type Module struct {
	Name string
	Fn   func()
	Deps []string
}

// Load 依据 Module.Deps 拓扑排序并发执行各模块。
//
// 加载前会校验依赖图，以下三种非法输入直接 panic：模块名重复、依赖了不存在
// 的模块、依赖成环（含自依赖）。成环若放任不管，相关 goroutine 会互相等待到
// 永久死锁、Load 永不返回，因此与另外两种错误一样尽早失败。
func Load(modules []Module) {
	names := make(map[string]bool, len(modules))
	for _, m := range modules {
		if names[m.Name] {
			panic("deploader: duplicate module name: " + m.Name)
		}
		names[m.Name] = true
	}

	for _, m := range modules {
		for _, dep := range m.Deps {
			if !names[dep] {
				panic(fmt.Sprintf("deploader: module %q depends on unknown module %q", m.Name, dep))
			}
		}
	}

	if cycle := findCycle(modules); len(cycle) > 0 {
		panic("deploader: cyclic dependency: " + strings.Join(cycle, " -> "))
	}

	done := make(map[string]chan struct{})
	for _, m := range modules {
		done[m.Name] = make(chan struct{})
	}

	var wg sync.WaitGroup
	for _, m := range modules {
		wg.Add(1)
		go func(m Module) {
			defer wg.Done()
			defer close(done[m.Name])

			for _, dep := range m.Deps {
				<-done[dep]
			}
			m.Fn()
		}(m)
	}
	wg.Wait()
}
