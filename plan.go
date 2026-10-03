package studyplan

import "sort"

// Generate computes the optimal learning plan for the given input.
//
// The plan maximizes the effective-gain vector ordered by skill
// priority (larger initial gap first, ties by skill ID) in
// lexicographic order; ties are broken by smaller total minutes.
// When no valuable plan fits the budget, an empty plan is returned.
//
// The search is exact (branch and bound); its worst-case cost is
// exponential in the number of incomplete courses, which suits the
// catalog sizes typical of a single employee's planning.
func Generate(in Input) Plan {
	if in.BudgetMinutes < 0 {
		in.BudgetMinutes = 0
	}

	completed := make(map[string]bool, len(in.Completed))
	for _, id := range in.Completed {
		completed[id] = true
	}

	// Candidates are incomplete courses, kept in catalog order.
	byID := make(map[string]Course, len(in.Catalog))
	var candidates []Course
	for _, c := range in.Catalog {
		byID[c.ID] = c
		if !completed[c.ID] {
			candidates = append(candidates, c)
		}
	}

	// skillsOf returns the deduplicated skill tags of a course.
	skillCache := make(map[string][]string, len(candidates))
	skillsOf := func(c Course) []string {
		if s, ok := skillCache[c.ID]; ok {
			return s
		}
		seen := make(map[string]bool, len(c.Skills))
		var uniq []string
		for _, s := range c.Skills {
			if !seen[s] {
				seen[s] = true
				uniq = append(uniq, s)
			}
		}
		skillCache[c.ID] = uniq
		return uniq
	}

	// topoOrders candidates so prerequisites come first. Prerequisite
	// edges to completed courses are already satisfied and ignored.
	topo := topoSort(candidates, byID, completed)

	var prio []prioSkill
	for skill, target := range in.Target {
		gap := target - in.Current[skill]
		if gap > 0 {
			prio = append(prio, prioSkill{skill, gap})
		}
	}
	sort.Slice(prio, func(i, j int) bool {
		if prio[i].gap != prio[j].gap {
			return prio[i].gap > prio[j].gap
		}
		return prio[i].id < prio[j].id
	})

	n := len(topo)
	if n == 0 || len(prio) == 0 {
		return Plan{Gains: buildGains(prio, nil)}
	}

	// idxOf maps course ID to its position in topo order.
	idxOf := make(map[string]int, n)
	for i, c := range topo {
		idxOf[c.ID] = i
	}

	// remaining[skill] lists topo positions of courses tagged with the
	// skill, sorted ascending, for upper-bound computation.
	remaining := make(map[string][]int)
	for i, c := range topo {
		for _, s := range skillsOf(c) {
			remaining[s] = append(remaining[s], i)
		}
	}

	increments := make(map[string]int)
	included := make([]bool, n)
	minutes := 0

	bestGains := make([]int, len(prio))
	bestMinutes := 0
	var bestSet []bool

	gainsNow := func() []int {
		g := make([]int, len(prio))
		for i, p := range prio {
			g[i] = min(p.gap, increments[p.id])
		}
		return g
	}

	// consider updates the best plan if the current selection wins.
	consider := func() {
		g := gainsNow()
		cmp := lexCompare(g, bestGains)
		if cmp > 0 || (cmp == 0 && (bestSet == nil || minutes < bestMinutes)) {
			bestGains = g
			bestMinutes = minutes
			bestSet = append([]bool(nil), included...)
		}
	}

	// boundCanWin reports whether any completion of the current prefix
	// could still beat the best plan found so far.
	boundCanWin := func(from int) bool {
		for i, p := range prio {
			extra := len(remaining[p.id]) - sort.SearchInts(remaining[p.id], from)
			bound := min(p.gap, increments[p.id]+extra)
			if bound > bestGains[i] {
				return true
			}
			if bound < bestGains[i] {
				return false
			}
		}
		return minutes < bestMinutes || bestSet == nil
	}

	var dfs func(i int)
	dfs = func(i int) {
		consider()
		if i == n || !boundCanWin(i) {
			return
		}
		c := topo[i]
		// Option 1: include course i, if all its incomplete
		// prerequisites are already included (they precede i in topo
		// order, so their decisions are final).
		if minutes+c.Minutes <= in.BudgetMinutes {
			ok := true
			for _, pre := range c.Prereqs {
				if completed[pre] {
					continue
				}
				j, isCandidate := idxOf[pre]
				if !isCandidate || !included[j] {
					ok = false
					break
				}
			}
			if ok {
				included[i] = true
				minutes += c.Minutes
				for _, s := range skillsOf(c) {
					increments[s]++
				}
				dfs(i + 1)
				for _, s := range skillsOf(c) {
					increments[s]--
				}
				minutes -= c.Minutes
				included[i] = false
			}
		}
		// Option 2: exclude course i.
		dfs(i + 1)
	}
	dfs(0)

	plan := Plan{Gains: buildGains(prio, bestGains)}
	if bestSet == nil {
		return plan
	}
	for i, c := range topo {
		if bestSet[i] {
			plan.Courses = append(plan.Courses, c.ID)
			plan.TotalMinutes += c.Minutes
		}
	}
	return plan
}

// topoSort orders candidate courses so that every incomplete
// prerequisite appears before its dependent. Ties follow catalog
// order for determinism.
func topoSort(candidates []Course, byID map[string]Course, completed map[string]bool) []Course {
	inSet := make(map[string]bool, len(candidates))
	for _, c := range candidates {
		inSet[c.ID] = true
	}
	indegree := make(map[string]int, len(candidates))
	dependents := make(map[string][]string)
	for _, c := range candidates {
		for _, pre := range c.Prereqs {
			if completed[pre] || !inSet[pre] {
				continue
			}
			indegree[c.ID]++
			dependents[pre] = append(dependents[pre], c.ID)
		}
	}
	var ready []string
	for _, c := range candidates {
		if indegree[c.ID] == 0 {
			ready = append(ready, c.ID)
		}
	}
	var order []Course
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, byID[id])
		for _, dep := range dependents[id] {
			indegree[dep]--
			if indegree[dep] == 0 {
				ready = append(ready, dep)
			}
		}
	}
	return order
}

// prioSkill is a skill with a positive initial gap.
type prioSkill struct {
	id  string
	gap int
}

// buildGains assembles the priority-ordered gain report. When gains
// is nil, all gains are zero.
func buildGains(prio []prioSkill, gains []int) []SkillGain {
	out := make([]SkillGain, len(prio))
	for i, p := range prio {
		out[i] = SkillGain{Skill: p.id, Gap: p.gap}
		if gains != nil {
			out[i].Gain = gains[i]
		}
	}
	return out
}

// lexCompare compares two gain vectors lexicographically.
func lexCompare(a, b []int) int {
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
