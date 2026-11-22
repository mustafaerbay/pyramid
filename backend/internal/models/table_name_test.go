package models

import "testing"

// TableName just pins each model to its schema.sql table — trivial, but a
// typo here would silently point GORM at the wrong table, so it's cheap
// insurance to assert every one explicitly in one place.
func TestTableNames(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"Person", Person{}.TableName(), "people"},
		{"Project", Project{}.TableName(), "projects"},
		{"ProjectMember", ProjectMember{}.TableName(), "project_members"},
		{"ProjectKickoff", ProjectKickoff{}.TableName(), "project_kickoffs"},
		{"KickoffActionItem", KickoffActionItem{}.TableName(), "kickoff_action_items"},
		{"Milestone", Milestone{}.TableName(), "milestones"},
		{"WeeklySummary", WeeklySummary{}.TableName(), "project_weekly_summaries"},
		{"Task", Task{}.TableName(), "tasks"},
		{"TaskAssignee", TaskAssignee{}.TableName(), "task_assignees"},
		{"Goal", Goal{}.TableName(), "goals"},
		{"LeaveRequest", LeaveRequest{}.TableName(), "leave_requests"},
		{"Contribution", Contribution{}.TableName(), "contributions"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s.TableName() = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}
