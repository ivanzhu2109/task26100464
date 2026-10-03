// Package studyplan generates an optimal in-memory learning plan:
// it closes an employee's largest job-skill gaps first within a fixed
// training-time budget, honoring course prerequisites.
package studyplan

// Course describes a single course in the catalog.
type Course struct {
	// ID uniquely identifies the course.
	ID string
	// Minutes is the positive integer duration of the course.
	Minutes int
	// Skills lists skills this course improves by one level each.
	// Duplicates are ignored.
	Skills []string
	// Prereqs lists course IDs that must be satisfied before this
	// course can be taken.
	Prereqs []string
}

// Input carries everything the planner needs, all in memory.
type Input struct {
	// Catalog is the full course catalog. References are guaranteed
	// complete and the prerequisite relation acyclic.
	Catalog []Course
	// Current maps skill ID to current proficiency (1-5).
	Current map[string]int
	// Target maps skill ID to job target proficiency (1-5).
	Target map[string]int
	// Completed lists IDs of courses the employee already finished.
	Completed []string
	// BudgetMinutes is the non-negative total time budget.
	BudgetMinutes int
}

// SkillGain reports the effective gain for one skill with a positive
// initial gap, ordered by planning priority.
type SkillGain struct {
	// Skill is the skill ID.
	Skill string
	// Gap is the initial gap (target minus current).
	Gap int
	// Gain is the effective gain, capped by Gap.
	Gain int
}

// Plan is the generated learning plan.
type Plan struct {
	// Courses lists selected course IDs, prerequisites first.
	Courses []string
	// TotalMinutes is the summed duration of selected courses.
	TotalMinutes int
	// Gains holds effective gains per skill, ordered by priority:
	// larger initial gap first, ties broken by skill ID.
	Gains []SkillGain
}
