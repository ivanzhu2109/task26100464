package learnplan

import "sort"

// GeneratePlan 根据请求生成最优学习计划。
//
// 在所有总分钟数不超过预算的合法计划中，使按技能优先级排列的
// 有效增益向量字典序最大；向量相同时取总分钟数最少者。
// 预算不足以安排任何有价值的学习时返回空计划。
func GeneratePlan(req Request) Plan {
	gaps := initialGaps(req)
	priorities := skillPriorities(gaps)
	if len(priorities) == 0 || req.Budget <= 0 {
		return Plan{Gains: map[string]int{}}
	}

	courses := schedulableCourses(req)
	best := search(courses, priorities, gaps, req.Budget)
	return best.toPlan(courses, priorities, gaps)
}

// initialGaps 计算各技能的正缺口（目标减当前，仅保留正值）。
func initialGaps(req Request) map[string]int {
	gaps := map[string]int{}
	for skill, target := range req.Target {
		if target < 1 {
			target = 1
		}
		if target > 5 {
			target = 5
		}
		current := req.Current[skill]
		if gap := target - current; gap > 0 {
			gaps[skill] = gap
		}
	}
	return gaps
}

// skillPriorities 按初始缺口从大到小、缺口相同按技能编号字典序排列，
// 生成过程中不改变。
func skillPriorities(gaps map[string]int) []string {
	skills := make([]string, 0, len(gaps))
	for skill := range gaps {
		skills = append(skills, skill)
	}
	sort.Slice(skills, func(i, j int) bool {
		if gaps[skills[i]] != gaps[skills[j]] {
			return gaps[skills[i]] > gaps[skills[j]]
		}
		return skills[i] < skills[j]
	})
	return skills
}

// schedulableCourses 过滤出未完成且先修可满足的课程，
// 并将先修解析为仅包含未完成课程的依赖下标。
func schedulableCourses(req Request) []courseNode {
	byID := make(map[string]Course, len(req.Catalog))
	for _, c := range req.Catalog {
		byID[c.ID] = c
	}

	// 未完成课程才需要安排；已完成课程直接满足先修，不追溯其先修。
	unfinished := map[string]bool{}
	for _, c := range req.Catalog {
		if !req.Completed[c.ID] {
			unfinished[c.ID] = true
		}
	}

	// 先修引用缺失且未完成的课程不可用，连同其下游一并剔除。
	usable := map[string]bool{}
	changed := true
	for changed {
		changed = false
		for id := range unfinished {
			if usable[id] {
				continue
			}
			ok := true
			for _, pre := range byID[id].Prereqs {
				if req.Completed[pre] {
					continue
				}
				if !unfinished[pre] || !usable[pre] {
					ok = false
					break
				}
			}
			if ok {
				usable[id] = true
				changed = true
			}
		}
	}

	index := map[string]int{}
	nodes := []courseNode{}
	for _, c := range req.Catalog {
		if !usable[c.ID] {
			continue
		}
		index[c.ID] = len(nodes)
		nodes = append(nodes, courseNode{course: c})
	}
	for i := range nodes {
		for _, pre := range nodes[i].course.Prereqs {
			if req.Completed[pre] {
				continue
			}
			if j, ok := index[pre]; ok {
				nodes[i].deps = append(nodes[i].deps, j)
			}
		}
	}
	return nodes
}
