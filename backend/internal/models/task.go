package models

import (
	"errors"
	"fmt"
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
// milestone, assignees, status, priority, due date.
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
	StartDate   *time.Time       `gorm:"type:date"`
	// EstimateManDays is the task's size as entered on the task itself;
	// used as the total until assignments carry man-days.
	EstimateManDays *float64   `gorm:"type:numeric(7,2)"`
	DueDate         *time.Time `gorm:"type:date"`
	CompletedAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time

	// AssigneeIDs holds the task's assignees (see the task_assignees join
	// table). It's not a column on this table — `gorm:"-"` keeps GORM from
	// trying to persist it directly — so the handler must populate it with a
	// separate task_assignees query (or a fresh write) before calling ToAPI.
	// Zero value is a nil slice; ToAPI normalizes that to `[]`, never `null`.
	AssigneeIDs []uuid.UUID `gorm:"-"`
	// ManDays is each assignee's share of the task's man-days, keyed by
	// person; loaded/written together with AssigneeIDs. Missing key = 0.
	ManDays map[uuid.UUID]float64 `gorm:"-"`
	// AvailableDays is each assignee's working days in [StartDate, DueDate]
	// outside approved leave (nil without both dates). Filled by the handler.
	AvailableDays map[uuid.UUID]*int `gorm:"-"`
	// Tags are the task's labels (labels/task_labels tables), sorted.
	// Filled by the handler; ToAPI turns nil into [].
	Tags []string `gorm:"-"`
}

// TableName pins the model to the existing pluralized table.
func (Task) TableName() string { return "tasks" }

// TaskAssignee is the GORM persistence model for the task_assignees join
// table — a task can have any number of assignees (replaces the earlier
// single assignee_id column on tasks).
type TaskAssignee struct {
	TaskID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	PersonID uuid.UUID `gorm:"type:uuid;primaryKey"`
	ManDays  float64   `gorm:"type:numeric(6,2);not null;default:0"`
}

func (TaskAssignee) TableName() string { return "task_assignees" }

// Label is a tag in the labels table. Tasks only use global labels
// (ProjectID nil), shared by every project.
type Label struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string     `gorm:"not null"`
	Color     string     `gorm:"not null;default:#888888"`
	ProjectID *uuid.UUID `gorm:"type:uuid"`
}

func (Label) TableName() string { return "labels" }

// TaskLabel links a task to a label.
type TaskLabel struct {
	TaskID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	LabelID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (TaskLabel) TableName() string { return "task_labels" }

// ToAPI converts the persistence model into the generated API DTO. The
// caller must have already populated t.AssigneeIDs (see the field comment).
func (t Task) ToAPI() api.Task {
	assigneeIDs := make([]openapi_types.UUID, len(t.AssigneeIDs))
	assignments := make([]api.TaskAssignment, len(t.AssigneeIDs))
	total := 0.0
	for i, id := range t.AssigneeIDs {
		assigneeIDs[i] = id
		md := t.ManDays[id]
		assignments[i] = api.TaskAssignment{PersonId: id, ManDays: md, AvailableDays: t.AvailableDays[id]}
		total += md
	}
	out := api.Task{
		Id:              t.ID,
		ProjectId:       t.ProjectID,
		Number:          t.Number,
		Title:           t.Title,
		Description:     t.Description,
		Status:          t.Status,
		Priority:        t.Priority,
		AssigneeIds:     assigneeIDs,
		Assignments:     assignments,
		ManDays:         total,
		EstimateManDays: t.EstimateManDays,
		MilestoneId:     t.MilestoneID,
		Tags:            t.Tags,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	if total == 0 && t.EstimateManDays != nil {
		out.ManDays = *t.EstimateManDays
	}
	if t.StartDate != nil {
		out.StartDate = &openapi_types.Date{Time: *t.StartDate}
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
// to log a task retroactively that's already in progress or done. Assignees
// aren't set here either — they need the task's generated ID first, so the
// handler inserts the task_assignees rows after the Create call succeeds.
func NewTaskFromCreate(in api.TaskCreate) Task {
	t := Task{
		ProjectID:   in.ProjectId,
		MilestoneID: in.MilestoneId,
		Title:       in.Title,
		Description: in.Description,
		Status:      api.TaskStatusBacklog,
		Priority:    api.TaskPriorityMedium,
	}
	if in.Status != nil {
		t.Status = *in.Status
	}
	if in.Priority != nil {
		t.Priority = *in.Priority
	}
	if in.StartDate != nil {
		d := in.StartDate.Time
		t.StartDate = &d
	}
	if in.EstimateManDays != nil {
		v := *in.EstimateManDays
		t.EstimateManDays = &v
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
// clearMilestone distinguishes "the request explicitly set milestoneId to
// null" (move to backlog) from "the request didn't mention the field at
// all" — both unmarshal a plain `*uuid.UUID` to nil, so the handler parses
// the raw JSON body once to tell them apart before calling this (see
// handlers.UpdateTask). Every other optional field in this app currently can
// only be set, not cleared, via PATCH; unassigning a task's milestone is
// common enough to be worth the extra plumbing here.
//
// Assignees are NOT handled here — in.AssigneeIds replaces the full
// task_assignees set for this task, which needs its own DB statements
// (delete-then-insert), so the handler manages that directly rather than
// through this in-memory mutation.
func (t *Task) ApplyUpdate(in api.TaskUpdate, clearMilestone bool) {
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
	if in.Priority != nil {
		t.Priority = *in.Priority
	}
	if in.StartDate != nil {
		d := in.StartDate.Time
		t.StartDate = &d
	}
	if in.EstimateManDays != nil {
		v := *in.EstimateManDays
		t.EstimateManDays = &v
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

// ValidateDates reports whether the planned start is after the due date,
// or the task-level estimate is out of range.
func (t Task) ValidateDates() error {
	if t.StartDate != nil && t.DueDate != nil && t.StartDate.After(*t.DueDate) {
		return errStartAfterDue
	}
	if t.EstimateManDays != nil && (*t.EstimateManDays < 0 || *t.EstimateManDays > MaxEstimateManDays) {
		return fmt.Errorf("estimateManDays must be between 0 and %d", MaxEstimateManDays)
	}
	return nil
}

// MaxEstimateManDays bounds a task's own man-day estimate.
const MaxEstimateManDays = 9999

var errStartAfterDue = errors.New("startDate must be on or before dueDate")

// MaxAssignmentManDays is the upper bound for one assignee's man-days on a
// task (matches the task_assignees.man_days CHECK constraint).
const MaxAssignmentManDays = 999

// ValidateAssignments checks every share is within [0, MaxAssignmentManDays].
func ValidateAssignments(in []api.TaskAssignment) error {
	for _, a := range in {
		if a.ManDays < 0 || a.ManDays > MaxAssignmentManDays {
			return fmt.Errorf("manDays must be between 0 and %d", MaxAssignmentManDays)
		}
	}
	return nil
}
