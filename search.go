package learnplan

// courseNode 是参与搜索的未完成课程及其未完成先修的下标依赖。
type courseNode struct {
	course Course
	deps   []int // 未完成先修在节点切片中的下标
}

// searchResult 是搜索得到的最优选择。
type searchResult struct {
	selected []int // 选中节点下标，按加入顺序（先修在前）
	minutes  int
	counts   map[string]int // 各技能累计提升次数（未按缺口截断）
}

// toPlan 把搜索结果转换为对外计划：技能有效增益按初始缺口截断。
// 选中顺序本身即满足先修在前。
func (r searchResult) toPlan(courses []courseNode, priorities []string, gaps map[string]int) Plan {
	plan := Plan{TotalMinutes: r.minutes, Gains: map[string]int{}}
	for _, idx := range r.selected {
		plan.Courses = append(plan.Courses, courses[idx].course)
	}
	for _, skill := range priorities {
		gain := r.counts[skill]
		if gain > gaps[skill] {
			gain = gaps[skill]
		}
		if gain > 0 {
			plan.Gains[skill] = gain
		}
	}
	return plan
}

// searcher 在“先修闭合”的课程子集空间上做分支限界搜索。
type searcher struct {
	nodes      []courseNode
	dependents [][]int // 每个节点的下游课程
	ready      []int   // 先修已全部入选、尚未决定的课程
	unmet      []int   // 每个节点尚未入选的未完成先修数

	alive      []bool         // 仍可能被选中（未入选且未被排除）
	aliveCount map[string]int // 各技能仍可能获得的提升次数
	counts     map[string]int // 已入选课程带来的各技能提升次数
	minutes    int
	selected   []int

	priorities []string
	gaps       map[string]int
	budget     int

	best    searchResult
	bestVec []int // 最优有效增益向量（按优先级排列）
}

// search 在所有先修闭合且不超预算的课程子集中，
// 搜索有效增益向量字典序最大、其次总分钟数最少的方案。
func search(courses []courseNode, priorities []string, gaps map[string]int, budget int) searchResult {
	s := &searcher{
		nodes:      courses,
		dependents: make([][]int, len(courses)),
		unmet:      make([]int, len(courses)),
		alive:      make([]bool, len(courses)),
		aliveCount: map[string]int{},
		counts:     map[string]int{},
		priorities: priorities,
		gaps:       gaps,
		budget:     budget,
		best:       searchResult{counts: map[string]int{}},
		bestVec:    make([]int, len(priorities)),
	}
	for i, n := range courses {
		s.unmet[i] = len(n.deps)
		for _, d := range n.deps {
			s.dependents[d] = append(s.dependents[d], i)
		}
		s.alive[i] = true
		for _, skill := range n.course.Skills {
			s.aliveCount[skill]++
		}
	}
	for i := range courses {
		if s.unmet[i] == 0 {
			s.ready = append(s.ready, i)
		}
	}
	s.dfs()
	return s.best
}

// dfs 处理当前就绪队列：对队首课程分别尝试“选”与“不选”。
func (s *searcher) dfs() {
	s.consider()
	if s.pruned() || len(s.ready) == 0 {
		return
	}

	head := s.ready[0]
	s.ready = s.ready[1:]

	// 分支一：选入 head（预算允许时）。
	if s.minutes+s.nodes[head].course.Minutes <= s.budget {
		unlocked := s.include(head)
		s.dfs()
		s.excludeIncluded(head, unlocked)
	}

	// 分支二：放弃 head，连带标记其不可达的下游。
	killed := s.kill(head)
	s.dfs()
	s.revive(killed)

	s.ready = append([]int{head}, s.ready...)
}

// include 选入节点，返回新就绪的节点列表（用于回溯）。
func (s *searcher) include(idx int) []int {
	c := s.nodes[idx].course
	s.minutes += c.Minutes
	s.selected = append(s.selected, idx)
	s.alive[idx] = false
	for _, skill := range c.Skills {
		s.counts[skill]++
		s.aliveCount[skill]--
	}
	var unlocked []int
	for _, d := range s.dependents[idx] {
		s.unmet[d]--
		if s.unmet[d] == 0 {
			s.ready = append(s.ready, d)
			unlocked = append(unlocked, d)
		}
	}
	return unlocked
}

// excludeIncluded 撤销 include。
func (s *searcher) excludeIncluded(idx int, unlocked []int) {
	c := s.nodes[idx].course
	for _, d := range unlocked {
		s.unmet[d]++
	}
	s.ready = s.ready[:len(s.ready)-len(unlocked)]
	for _, skill := range c.Skills {
		s.counts[skill]--
		s.aliveCount[skill]++
	}
	s.alive[idx] = true
	s.selected = s.selected[:len(s.selected)-1]
	s.minutes -= c.Minutes
}

// kill 排除节点并级联标记下游不可达，返回本次置死的节点。
func (s *searcher) kill(idx int) []int {
	var killed []int
	stack := []int{idx}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !s.alive[cur] {
			continue
		}
		s.alive[cur] = false
		killed = append(killed, cur)
		for _, skill := range s.nodes[cur].course.Skills {
			s.aliveCount[skill]--
		}
		stack = append(stack, s.dependents[cur]...)
	}
	return killed
}

// revive 撤销 kill。
func (s *searcher) revive(killed []int) {
	for _, idx := range killed {
		s.alive[idx] = true
		for _, skill := range s.nodes[idx].course.Skills {
			s.aliveCount[skill]++
		}
	}
}

// effectiveVec 计算当前已选集合的有效增益向量（按优先级、按缺口截断）。
func (s *searcher) effectiveVec() []int {
	vec := make([]int, len(s.priorities))
	for i, skill := range s.priorities {
		vec[i] = s.counts[skill]
		if vec[i] > s.gaps[skill] {
			vec[i] = s.gaps[skill]
		}
	}
	return vec
}

// boundVec 计算乐观上界：假设所有仍可选的课程最终都入选。
func (s *searcher) boundVec() []int {
	vec := make([]int, len(s.priorities))
	for i, skill := range s.priorities {
		vec[i] = s.counts[skill] + s.aliveCount[skill]
		if vec[i] > s.gaps[skill] {
			vec[i] = s.gaps[skill]
		}
	}
	return vec
}

// consider 在当前合法解上更新最优结果。
func (s *searcher) consider() {
	vec := s.effectiveVec()
	switch compareVec(vec, s.bestVec) {
	case 1:
		s.saveBest(vec)
	case 0:
		if s.minutes < s.best.minutes {
			s.saveBest(vec)
		}
	}
}

func (s *searcher) saveBest(vec []int) {
	s.bestVec = vec
	s.best = searchResult{
		selected: append([]int(nil), s.selected...),
		minutes:  s.minutes,
		counts:   copyCounts(s.counts),
	}
}

// pruned 判断当前分支是否不可能产生更优结果。
func (s *searcher) pruned() bool {
	switch compareVec(s.boundVec(), s.bestVec) {
	case -1:
		return true
	case 0:
		// 增益向量无法更优，且继续选只会增加分钟数。
		return s.minutes >= s.best.minutes
	}
	return false
}

// compareVec 字典序比较两个等长向量：a 大返回 1，b 大返回 -1，相等返回 0。
func compareVec(a, b []int) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] > b[i] {
				return 1
			}
			return -1
		}
	}
	return 0
}

func copyCounts(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
