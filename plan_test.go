package studyplan

import (
	"reflect"
	"testing"
)

func gainsBySkill(p Plan) map[string]int {
	m := make(map[string]int, len(p.Gains))
	for _, g := range p.Gains {
		m[g.Skill] = g.Gain
	}
	return m
}

// 共享先修使“局部最便宜”方案不再最优：为 s1 单独选最便宜的 C1 后，
// 预算不足以再补 s2；改走共享先修 P 反而能用同样预算同时补上两项技能。
func TestSharedPrereqBeatsLocallyCheapest(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "C1", Minutes: 50, Skills: []string{"s1"}},
			{ID: "P", Minutes: 30},
			{ID: "A", Minutes: 40, Skills: []string{"s1"}, Prereqs: []string{"P"}},
			{ID: "B", Minutes: 40, Skills: []string{"s2"}, Prereqs: []string{"P"}},
		},
		Current:       map[string]int{"s1": 1, "s2": 1},
		Target:        map[string]int{"s1": 2, "s2": 2},
		BudgetMinutes: 110,
	}
	plan := Generate(in)

	wantCourses := []string{"P", "A", "B"}
	if !reflect.DeepEqual(plan.Courses, wantCourses) {
		t.Errorf("Courses = %v, want %v", plan.Courses, wantCourses)
	}
	if plan.TotalMinutes != 110 {
		t.Errorf("TotalMinutes = %d, want 110", plan.TotalMinutes)
	}
	gains := gainsBySkill(plan)
	if gains["s1"] != 1 || gains["s2"] != 1 {
		t.Errorf("gains = %v, want s1=1 s2=1", gains)
	}
}

// 已完成课程直接满足先修条件，不追溯补学它自己的先修课。
func TestCompletedCourseTruncatesPrereqTracing(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "Old", Minutes: 10},
			{ID: "Done", Minutes: 20, Skills: []string{"s1"}, Prereqs: []string{"Old"}},
			{ID: "T", Minutes: 30, Skills: []string{"s1"}, Prereqs: []string{"Done"}},
		},
		Current:       map[string]int{"s1": 1},
		Target:        map[string]int{"s1": 2},
		Completed:     []string{"Done"},
		BudgetMinutes: 100,
	}
	plan := Generate(in)

	want := []string{"T"}
	if !reflect.DeepEqual(plan.Courses, want) {
		t.Errorf("Courses = %v, want %v (Old must not be traced, Done must not repeat)", plan.Courses, want)
	}
	if plan.TotalMinutes != 30 {
		t.Errorf("TotalMinutes = %d, want 30", plan.TotalMinutes)
	}
	if gains := gainsBySkill(plan); gains["s1"] != 1 {
		t.Errorf("s1 gain = %d, want 1", gains["s1"])
	}
}

// 预算不足以安排任何有价值的学习时返回空计划。
func TestEmptyPlanWhenBudgetTooSmall(t *testing.T) {
	in := Input{
		Catalog:       []Course{{ID: "C1", Minutes: 60, Skills: []string{"s1"}}},
		Current:       map[string]int{"s1": 1},
		Target:        map[string]int{"s1": 3},
		BudgetMinutes: 30,
	}
	plan := Generate(in)
	if len(plan.Courses) != 0 || plan.TotalMinutes != 0 {
		t.Errorf("plan = %+v, want empty", plan)
	}
	if len(plan.Gains) != 1 || plan.Gains[0].Gain != 0 || plan.Gains[0].Gap != 2 {
		t.Errorf("Gains = %+v, want one zero-gain entry with gap 2", plan.Gains)
	}
}

// 有效增益不超过初始缺口；增益相同选总分钟数更少的计划。
func TestGainCappedByGapAndMinMinutesTieBreak(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "Cheap", Minutes: 40, Skills: []string{"s1"}},
			{ID: "Pricey", Minutes: 50, Skills: []string{"s1"}},
		},
		Current:       map[string]int{"s1": 4},
		Target:        map[string]int{"s1": 5},
		BudgetMinutes: 200,
	}
	plan := Generate(in)
	if !reflect.DeepEqual(plan.Courses, []string{"Cheap"}) {
		t.Errorf("Courses = %v, want [Cheap]", plan.Courses)
	}
	if plan.TotalMinutes != 40 {
		t.Errorf("TotalMinutes = %d, want 40", plan.TotalMinutes)
	}
	if gains := gainsBySkill(plan); gains["s1"] != 1 {
		t.Errorf("s1 gain = %d, want 1 (capped by gap)", gains["s1"])
	}
}

// 较高优先级技能多提升 1 级，优于任意数量的较低优先级提升。
func TestHigherPriorityDominates(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "Low1", Minutes: 10, Skills: []string{"low"}},
			{ID: "Low2", Minutes: 10, Skills: []string{"low"}},
			{ID: "High", Minutes: 100, Skills: []string{"high"}},
		},
		Current:       map[string]int{"high": 1, "low": 4},
		Target:        map[string]int{"high": 4, "low": 5},
		BudgetMinutes: 100,
	}
	plan := Generate(in)
	if !reflect.DeepEqual(plan.Courses, []string{"High"}) {
		t.Errorf("Courses = %v, want [High]", plan.Courses)
	}
	// 优先级按初始缺口排序：high 缺口 3 应排在 low 缺口 1 之前。
	if plan.Gains[0].Skill != "high" || plan.Gains[1].Skill != "low" {
		t.Errorf("priority order = [%s %s], want [high low]", plan.Gains[0].Skill, plan.Gains[1].Skill)
	}
}

// 高优先级由缺口决定：缺口大的技能优先，即使它会挤掉多个小缺口技能。
func TestLargerGapWinsOverManySmallGains(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "Big", Minutes: 90, Skills: []string{"big"}},
			{ID: "Small1", Minutes: 40, Skills: []string{"s1"}},
			{ID: "Small2", Minutes: 40, Skills: []string{"s2"}},
		},
		Current:       map[string]int{"big": 1, "s1": 4, "s2": 4},
		Target:        map[string]int{"big": 3, "s1": 5, "s2": 5},
		BudgetMinutes: 90,
	}
	plan := Generate(in)
	if !reflect.DeepEqual(plan.Courses, []string{"Big"}) {
		t.Errorf("Courses = %v, want [Big]", plan.Courses)
	}
}

// 先修课必须先于后续课输出，且共享先修只出现一次、计时一次、贡献一次。
func TestPrereqOrderingAndSharedCountedOnce(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "A", Minutes: 40, Skills: []string{"s1"}, Prereqs: []string{"P"}},
			{ID: "B", Minutes: 40, Skills: []string{"s2"}, Prereqs: []string{"P"}},
			{ID: "P", Minutes: 30, Skills: []string{"s3"}},
		},
		Current:       map[string]int{"s1": 1, "s2": 1, "s3": 1},
		Target:        map[string]int{"s1": 2, "s2": 2, "s3": 2},
		BudgetMinutes: 110,
	}
	plan := Generate(in)
	if !reflect.DeepEqual(plan.Courses, []string{"P", "A", "B"}) {
		t.Errorf("Courses = %v, want [P A B]", plan.Courses)
	}
	if plan.TotalMinutes != 110 {
		t.Errorf("TotalMinutes = %d, want 110 (P counted once)", plan.TotalMinutes)
	}
	gains := gainsBySkill(plan)
	if gains["s1"] != 1 || gains["s2"] != 1 || gains["s3"] != 1 {
		t.Errorf("gains = %v, want s1=1 s2=1 s3=1 (P contributes once)", gains)
	}
}

// 一门课程的所有先修都必须满足，先修链要完整纳入计划。
func TestTransitivePrereqChainIncluded(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "C", Minutes: 10, Skills: []string{"s1"}, Prereqs: []string{"B"}},
			{ID: "B", Minutes: 10, Prereqs: []string{"A"}},
			{ID: "A", Minutes: 10},
		},
		Current:       map[string]int{"s1": 1},
		Target:        map[string]int{"s1": 2},
		BudgetMinutes: 30,
	}
	plan := Generate(in)
	if !reflect.DeepEqual(plan.Courses, []string{"A", "B", "C"}) {
		t.Errorf("Courses = %v, want [A B C]", plan.Courses)
	}
}

// 熟练度上限为 5：缺口本身已限制有效增益，累计提升不会突破。
func TestLevelCapAtFive(t *testing.T) {
	in := Input{
		Catalog: []Course{
			{ID: "C1", Minutes: 10, Skills: []string{"s1"}},
			{ID: "C2", Minutes: 10, Skills: []string{"s1"}},
		},
		Current:       map[string]int{"s1": 4},
		Target:        map[string]int{"s1": 5},
		BudgetMinutes: 20,
	}
	plan := Generate(in)
	if gains := gainsBySkill(plan); gains["s1"] != 1 {
		t.Errorf("s1 gain = %d, want 1 (capped at level 5)", gains["s1"])
	}
}

// 无正缺口时返回空计划。
func TestNoGapReturnsEmpty(t *testing.T) {
	in := Input{
		Catalog:       []Course{{ID: "C1", Minutes: 10, Skills: []string{"s1"}}},
		Current:       map[string]int{"s1": 5},
		Target:        map[string]int{"s1": 5},
		BudgetMinutes: 100,
	}
	plan := Generate(in)
	if len(plan.Courses) != 0 || len(plan.Gains) != 0 {
		t.Errorf("plan = %+v, want empty", plan)
	}
}
