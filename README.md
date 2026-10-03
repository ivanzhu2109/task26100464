# learnplan

学习计划生成库（Go，纯内存计算，无外部依赖）。在固定培训时长内，
为经理选出优先补齐员工最严重岗位技能缺口的最优课程组合。

## 规则概要

- 技能优先级：按初始缺口（目标 − 当前，仅正缺口）从大到小排列，
  缺口相同按技能编号字典序排列，生成过程中不变。
- 每门未完成课程使其每个技能标签提升 1 级（上限 5），可累计；
  比较计划价值时，每项技能的有效增益最多等于其初始缺口。
- 已完成课程不再安排、不再加技能，但可直接满足先修条件，
  不追溯补学它自己的先修课。
- 计划自动包含所选课程的全部未完成先修课，并按先修在前输出；
  共享课程只出现一次、计时一次、贡献技能一次。
- 在所有不超预算的合法计划中，按技能优先级排列的有效增益向量
  字典序最大者胜出（高优先级多 1 级优于任意数量低优先级提升）；
  向量相同取总分钟数最少者；预算不足时返回空计划。

## 使用

```go
package main

import (
	"fmt"

	"learnplan"
)

func main() {
	plan := learnplan.GeneratePlan(learnplan.Request{
		Catalog: []learnplan.Course{
			{ID: "go-basics", Minutes: 60, Skills: []string{"go"}},
			{ID: "go-advanced", Minutes: 90, Skills: []string{"go", "testing"},
				Prereqs: []string{"go-basics"}},
		},
		Current:   map[string]int{"go": 2, "testing": 3},
		Target:    map[string]int{"go": 4, "testing": 4},
		Completed: map[string]bool{},
		Budget:    150,
	})

	for _, c := range plan.Courses {
		fmt.Println(c.ID, c.Minutes)
	}
	fmt.Println("total:", plan.TotalMinutes, "gains:", plan.Gains)
}
```

## API

- `Course`：`ID`、`Minutes`（正整数分钟）、`Skills`（去重技能标签）、`Prereqs`（先修课程编号）。
- `Request`：`Catalog`、`Current`、`Target`（熟练度均为 1–5）、
  `Completed`（已完成课程编号集合）、`Budget`（非负整数分钟）。
- `Plan`：`Courses`（先修在前）、`TotalMinutes`、`Gains`（各技能有效增益）。
- `GeneratePlan(req Request) Plan`：同步、纯内存、无 goroutine 泄漏，可直接并发调用（各次调用间无共享状态）。

## 实现说明

搜索层（`search.go`）在“先修闭合”的课程子集空间上做分支限界 DFS：
维护先修已满足的就绪队列，对队首课程分别尝试选/不选；
用“所有仍可选课程都入选”的乐观增益向量做上界剪枝，
增益向量无法更优且分钟数无法更少时提前返回。

## 测试

```sh
go test ./...
```

覆盖：共享先修使局部最便宜方案不再最优、已完成课程截断先修追溯、
预算不足返回空计划、高优先级字典序占优、增益按缺口截断与分钟数
平局裁决、多技能课程只计一次。
