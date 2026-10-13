package deploader

import "slices"

// findCycle 返回依赖图中的一条环（形如 [a b a]），无环时返回 nil。
//
// 深度优先遍历：若当前节点已存在于本次遍历路径中，说明找到了环，
// 路径中从该节点到栈顶的部分补上该节点自身即为环。
func findCycle(modules []Module) []string {
	deps := make(map[string][]string, len(modules))
	for _, m := range modules {
		deps[m.Name] = m.Deps
	}

	visited := make(map[string]bool, len(modules))
	inPath := make(map[string]int, len(modules))
	var stack []string

	var visit func(name string) []string
	visit = func(name string) []string {
		if visited[name] {
			return nil
		}
		if idx, ok := inPath[name]; ok {
			// 回到本次路径上的节点：截取环并补回起点。
			return append(slices.Clone(stack[idx:]), name)
		}

		inPath[name] = len(stack)
		stack = append(stack, name)

		for _, dep := range deps[name] {
			if cycle := visit(dep); cycle != nil {
				return cycle
			}
		}

		stack = stack[:len(stack)-1]
		delete(inPath, name)
		visited[name] = true
		return nil
	}

	for _, m := range modules {
		if cycle := visit(m.Name); cycle != nil {
			return cycle
		}
	}
	return nil
}
