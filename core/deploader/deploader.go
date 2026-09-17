package deploader

import (
	"fmt"
	"sync"
)

// Module 定义一个可加载的依赖模块
type Module struct {
	Name string
	Fn   func()
	Deps []string
}

// Load 依据 Module.Deps 拓扑排序并发执行各模块，校验依赖名合法性
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
