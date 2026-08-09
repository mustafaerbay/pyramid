package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// Task is the GORM persistence model for the `tasks` table defined in
// db/schema.sql. The full schema also has type/sprint/parent/story-points/
// estimate/reporter/dependencies/comments/labels columns — none of those are
// wired up yet (flagged as future work); this model covers the breakdown-
// into-assignable-work slice the project Tasks section needs: title,
// milestone, assignee, status, priority, due date.
type Task struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null;index"`
	// Number is the per-project sequential number (e.g. 42 for "PYR-42"),
	// assigned by the handler — see handlers.nextTaskNumber.
	Number      int              `gorm:"not null"`
	MilestoneID *uuid.UUID       `gorm:"type:uuid;index"`
	Title       string           `gorm:"not null"`
	Description *string          `gorm:"type:text"`
	Status      api.TaskStatus   `gorm:"type:task_status;not null;default:backlog"`
	Priority    api.TaskPriority `gorm:"type:task_priority;not null;default:medium"`
	AssigneeID  *uuid.UUID       `gorm:"type:uuid;index"`
	DueDate     *time.Time       `gorm:"type:date"`
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName pins the model to the existing pluralized table.
func (Task) TableName() string { return "tasks" }

// ToAPI converts the persistence model into the generated API DTO.
func (t Task) ToAPI() api.Task {
	out := api.Task{
		Id:          t.ID,
		ProjectId:   t.ProjectID,
		Number:      t.Number,
		Title:       t.Title,
		Description: t.Description,
		Status:      t.Status,
		Priority:    t.Priority,
		AssigneeId:  t.AssigneeID,
		MilestoneId: t.MilestoneID,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
	if t.DueDate != nil {
		out.DueDate = &openapi_types.Date{Time: *t.DueDate}
	}
	if t.CompletedAt != nil {
		out.CompletedAt = t.CompletedAt
	}
	return out
}

// NewTaskFromCreate builds a new Task from a create request. Number is left
// zero-valued — the handler assigns the next per-project sequential number
// before inserting, since that requires a DB query this constructor
// shouldn't own. Status/priority default the same way the DB columns do
// (backlog/medium) when the request doesn't specify them, but — unlike
// Milestone/Goal — a task's initial status IS caller-settable: it's normal
// to log a task retroactively that's already in progress or done.
func NewTaskFromCreate(in api.TaskCreate) Task {
	t := Task{
		ProjectID:   in.ProjectId,
		MilestoneID: in.MilestoneId,
		Title:       in.Title,
		Description: in.Description,
		AssigneeID:  in.AssigneeId,
		Status:      api.TaskStatusBacklog,
		Priority:    api.TaskPriorityMedium,
	}
	if in.Status != nil {
		t.Status = *in.Status
	}
	if in.Priority != nil {
		t.Priority = *in.Priority
	}
	if in.DueDate != nil {
		d := in.DueDate.Time
		t.DueDate = &d
	}
	if t.Status == api.TaskStatusDone {
		now := time.Now().UTC()
		t.CompletedAt = &now
	}
	return t
}

// ApplyUpdate mutates the model in place with the fields of a patch.
// clearMilestone/clearAssignee distinguish "the request explicitly set this
// field to null" (unassign / move to backlog) from "the request didn't
// mention this field at all" — both unmarshal a plain `*uuid.UUID` to nil,
// so the handler parses the raw JSON body once to tell them apart before
// calling this (see handlers.UpdateTask). Every other optional field in this
// app currently can only be set, not cleared, via PATCH; unassigning a task
// is common enough to be worth the extra plumbing here.
func (t *Task) ApplyUpdate(in api.TaskUpdate, clearMilestone, clearAssignee bool) {
	if in.Title != nil {
		t.Title = *in.Title
	}
	if in.Description != nil {
		t.Description = in.Description
	}
	if clearMilestone {
		t.MilestoneID = nil
	} else if in.MilestoneId != nil {
		t.MilestoneID = in.MilestoneId
	}
	if clearAssignee {
		t.AssigneeID = nil
	} else if in.AssigneeId != nil {
		t.AssigneeID = in.AssigneeId
	}
	if in.Priority != nil {
		t.Priority = *in.Priority
	}
	if in.DueDate != nil {
		d := in.DueDate.Time
		t.DueDate = &d
	}
	if in.Status != nil {
		t.Status = *in.Status
		// Marking a task done stamps "now" automatically, unless one was
		// already set (e.g. it was already done and this patch changes
		// something else). Moving a task OUT of done leaves the old
		// completedAt in place rather than clearing it — same "don't
		// silently lose history" rationale as Milestone.
		if t.Status == api.TaskStatusDone && t.CompletedAt == nil {
			now := time.Now().UTC()
			t.CompletedAt = &now
		}
	}
}
