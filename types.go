// Package learnplan 生成固定培训时长内、按技能优先级字典序最优的学习计划。
package learnplan

// Course 描述目录中的一门课程。
type Course struct {
	ID      string   // 课程唯一编号
	Minutes int      // 正整数分钟数
	Skills  []string // 去重后的技能标签
	Prereqs []string // 先修课程编号
}

// Request 是一次学习计划生成的全部输入，全部在内存中提供。
type Request struct {
	Catalog   []Course        // 课程目录（引用完整、先修无环）
	Current   map[string]int  // 技能当前熟练度（1-5），已包含历史学习结果
	Target    map[string]int  // 岗位目标熟练度（1-5）
	Completed map[string]bool // 已完成课程编号集合
	Budget    int             // 非负整数分钟预算
}

// Plan 是生成的学习计划。
type Plan struct {
	Courses      []Course       // 需学习的课程，先修在前
	TotalMinutes int            // 总分钟数，不超过预算
	Gains        map[string]int // 各技能的有效增益（不超过初始缺口）
}
