package learnplan

import (
	"reflect"
	"testing"
)

func planIDs(p Plan) []string {
	ids := make([]string, 0, len(p.Courses))
	for _, c := range p.Courses {
		ids = append(ids, c.ID)
	}
	return ids
}

// 共享先修使“局部最便宜”方案不再最优：
// 单看 a（含先修 p 共 50 分钟）不如直接买 c（25 分钟）划算，
// 但 p 同时是 b 的先修，p+a+b 共享一次先修成本后整体最优。
func TestSharedPrereqBeatsLocalCheapest(t *testing.T) {
	req := Request{
		Catalog: []Course{
			{ID: "p", Minutes: 40},
			{ID: "a", Minutes: 10, Skills: []string{"s1"}, Prereqs: []string{"p"}},
			{ID: "b", Minutes: 10, Skills: []string{"s2"}, Prereqs: []string{"p"}},
			{ID: "c", Minutes: 25, Skills: []string{"s1"}},
		},
		Current: map[string]int{"s1": 4, "s2": 4},
		Target:  map[string]int{"s1": 5, "s2": 5},
		Budget:  60,
	}
	plan := GeneratePlan(req)

	if plan.TotalMinutes != 60 {
		t.Fatalf("TotalMinutes = %d, want 60", plan.TotalMinutes)
	}
	if got := planIDs(plan); got[0] != "p" || len(got) != 3 {
		t.Fatalf("plan order = %v, want p first with 3 courses", got)
	}
	got := map[string]bool{}
	for _, id := range planIDs(plan) {
		got[id] = true
	}
	if !got["a"] || !got["b"] || got["c"] {
		t.Fatalf("plan = %v, want {p,a,b}", planIDs(plan))
	}
	if !reflect.DeepEqual(plan.Gains, map[string]int{"s1": 1, "s2": 1}) {
		t.Fatalf("Gains = %v, want s1:1 s2:1", plan.Gains)
	}
}

// 已完成课程直接满足先修，不追溯补学它自己的先修课。
func TestCompletedCourseTruncatesPrereqChain(t *testing.T) {
	req := Request{
		Catalog: []Course{
			{ID: "p0", Minutes: 100, Skills: []string{"s1"}},
			{ID: "done", Minutes: 5, Skills: []string{"s1"}, Prereqs: []string{"p0"}},
			{ID: "c", Minutes: 10, Skills: []string{"s1"}, Prereqs: []string{"done"}},
		},
		Current:   map[string]int{"s1": 2},
		Target:    map[string]int{"s1": 3},
		Completed: map[string]bool{"done": true},
		Budget:    10,
	}
	plan := GeneratePlan(req)

	if got := planIDs(plan); !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("plan = %v, want [c] only", got)
	}
	if plan.TotalMinutes != 10 {
		t.Fatalf("TotalMinutes = %d, want 10", plan.TotalMinutes)
	}
	if !reflect.DeepEqual(plan.Gains, map[string]int{"s1": 1}) {
		t.Fatalf("Gains = %v, want s1:1", plan.Gains)
	}
}

// 预算不足以安排任何有价值的学习时返回空计划。
func TestEmptyPlanWhenBudgetTooSmall(t *testing.T) {
	req := Request{
		Catalog: []Course{{ID: "x", Minutes: 10, Skills: []string{"s1"}}},
		Current: map[string]int{"s1": 1},
		Target:  map[string]int{"s1": 3},
		Budget:  5,
	}
	plan := GeneratePlan(req)
	if len(plan.Courses) != 0 || plan.TotalMinutes != 0 || len(plan.Gains) != 0 {
		t.Fatalf("plan = %+v, want empty", plan)
	}
}

// 高优先级技能多提升 1 级，优于任意数量的低优先级提升。
func TestHigherPriorityDominates(t *testing.T) {
	req := Request{
		Catalog: []Course{
			{ID: "x", Minutes: 10, Skills: []string{"s1"}},
			{ID: "y", Minutes: 10, Skills: []string{"s2"}},
			{ID: "z", Minutes: 10, Skills: []string{"s2"}},
		},
		Current: map[string]int{"s1": 1, "s2": 2},
		Target:  map[string]int{"s1": 5, "s2": 5},
		Budget:  20,
	}
	plan := GeneratePlan(req)
	// s1 缺口 4 优先于 s2 缺口 3；(1,1) 优于 (0,2)：必须包含 x。
	ids := map[string]bool{}
	for _, id := range planIDs(plan) {
		ids[id] = true
	}
	if !ids["x"] {
		t.Fatalf("plan = %v, want x included", planIDs(plan))
	}
	if !reflect.DeepEqual(plan.Gains, map[string]int{"s1": 1, "s2": 1}) {
		t.Fatalf("Gains = %v, want s1:1 s2:1", plan.Gains)
	}
}

// 有效增益按初始缺口截断；向量相同时选总分钟数最少。
func TestGainCappedAndMinMinutesTieBreak(t *testing.T) {
	req := Request{
		Catalog: []Course{
			{ID: "u", Minutes: 5, Skills: []string{"s1"}},
			{ID: "v", Minutes: 8, Skills: []string{"s1"}},
		},
		Current: map[string]int{"s1": 4},
		Target:  map[string]int{"s1": 5},
		Budget:  13,
	}
	plan := GeneratePlan(req)
	if got := planIDs(plan); !reflect.DeepEqual(got, []string{"u"}) {
		t.Fatalf("plan = %v, want [u]", got)
	}
	if plan.TotalMinutes != 5 {
		t.Fatalf("TotalMinutes = %d, want 5", plan.TotalMinutes)
	}
	if !reflect.DeepEqual(plan.Gains, map[string]int{"s1": 1}) {
		t.Fatalf("Gains = %v, want s1:1", plan.Gains)
	}
}

// 一门课程提升多项技能，且共享课程只出现一次、计时一次。
func TestMultiSkillCourseCountedOnce(t *testing.T) {
	req := Request{
		Catalog: []Course{
			{ID: "m", Minutes: 12, Skills: []string{"s1", "s2"}},
			{ID: "n", Minutes: 12, Skills: []string{"s2"}, Prereqs: []string{"m"}},
		},
		Current: map[string]int{"s1": 4, "s2": 3},
		Target:  map[string]int{"s1": 5, "s2": 5},
		Budget:  24,
	}
	plan := GeneratePlan(req)
	if got := planIDs(plan); !reflect.DeepEqual(got, []string{"m", "n"}) {
		t.Fatalf("plan = %v, want [m n]", got)
	}
	if plan.TotalMinutes != 24 {
		t.Fatalf("TotalMinutes = %d, want 24", plan.TotalMinutes)
	}
	if !reflect.DeepEqual(plan.Gains, map[string]int{"s1": 1, "s2": 2}) {
		t.Fatalf("Gains = %v, want s1:1 s2:2", plan.Gains)
	}
}
