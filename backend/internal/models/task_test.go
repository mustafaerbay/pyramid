package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestNewTaskFromCreate_DefaultsToBacklogMedium(t *testing.T) {
	task := NewTaskFromCreate(api.TaskCreate{ProjectId: uuid.New(), Title: "Wire up auth"})

	if task.Status != api.TaskStatusBacklog {
		t.Errorf("Status = %q, want backlog (default)", task.Status)
	}
	if task.Priority != api.TaskPriorityMedium {
		t.Errorf("Priority = %q, want medium (default)", task.Priority)
	}
	if task.CompletedAt != nil {
		t.Errorf("CompletedAt = %v, want nil for a non-done task", task.CompletedAt)
	}
}

func TestNewTaskFromCreate_UnlikeMilestoneStatusIsCallerSettable(t *testing.T) {
	// A task can be logged retroactively as already done — this is the one
	// place this app's "always starts pending/backlog" convention doesn't
	// apply (see the doc comment on NewTaskFromCreate).
	done := api.TaskStatusDone
	task := NewTaskFromCreate(api.TaskCreate{ProjectId: uuid.New(), Title: "Retroactive work", Status: &done})

	if task.Status != api.TaskStatusDone {
		t.Errorf("Status = %q, want done", task.Status)
	}
	if task.CompletedAt == nil {
		t.Error("CompletedAt = nil, want auto-stamped since the task was created already done")
	}
}

func TestTask_ApplyUpdate_MilestoneClearVsUntouched(t *testing.T) {
	milestoneID := uuid.New()
	otherMilestoneID := uuid.New()

	t.Run("clearMilestone=true clears it regardless of MilestoneId in the patch", func(t *testing.T) {
		task := Task{MilestoneID: &milestoneID}
		task.ApplyUpdate(api.TaskUpdate{}, true)
		if task.MilestoneID != nil {
			t.Errorf("MilestoneID = %v, want nil", task.MilestoneID)
		}
	})

	t.Run("clearMilestone=false with MilestoneId set moves it", func(t *testing.T) {
		task := Task{MilestoneID: &milestoneID}
		task.ApplyUpdate(api.TaskUpdate{MilestoneId: &otherMilestoneID}, false)
		if task.MilestoneID == nil || *task.MilestoneID != otherMilestoneID {
			t.Errorf("MilestoneID = %v, want %v", task.MilestoneID, otherMilestoneID)
		}
	})

	t.Run("clearMilestone=false with MilestoneId nil leaves it untouched", func(t *testing.T) {
		task := Task{MilestoneID: &milestoneID}
		task.ApplyUpdate(api.TaskUpdate{}, false)
		if task.MilestoneID == nil || *task.MilestoneID != milestoneID {
			t.Errorf("MilestoneID = %v, want unchanged %v", task.MilestoneID, milestoneID)
		}
	})
}

func TestTask_ApplyUpdate_MarkingDoneStampsCompletedAtOnce(t *testing.T) {
	task := Task{Status: api.TaskStatusInProgress}

	done := api.TaskStatusDone
	task.ApplyUpdate(api.TaskUpdate{Status: &done}, false)
	if task.CompletedAt == nil {
		t.Fatal("CompletedAt = nil, want auto-stamped")
	}
	firstStamp := *task.CompletedAt

	// Moving status again (still done, e.g. an unrelated field changes)
	// must not overwrite the original completion timestamp.
	task.ApplyUpdate(api.TaskUpdate{Status: &done}, false)
	if !task.CompletedAt.Equal(firstStamp) {
		t.Errorf("CompletedAt changed to %v, want unchanged %v", task.CompletedAt, firstStamp)
	}
}

func TestTask_ApplyUpdate_MovingOutOfDoneKeepsOldCompletedAt(t *testing.T) {
	original := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := Task{Status: api.TaskStatusDone, CompletedAt: &original}

	inReview := api.TaskStatusInReview
	task.ApplyUpdate(api.TaskUpdate{Status: &inReview}, false)

	if task.Status != api.TaskStatusInReview {
		t.Errorf("Status = %q, want in_review", task.Status)
	}
	if task.CompletedAt == nil || !task.CompletedAt.Equal(original) {
		t.Errorf("CompletedAt = %v, want preserved %v (don't silently lose history)", task.CompletedAt, original)
	}
}

func TestTask_ApplyUpdate_AssigneesAreNotHandledHere(t *testing.T) {
	// TaskUpdate.AssigneeIds is documented as handled by the handler, not
	// ApplyUpdate (needs its own delete-then-insert DB statements). This
	// test pins that contract: ApplyUpdate must never touch AssigneeIDs.
	ids := []uuid.UUID{uuid.New()}
	task := Task{AssigneeIDs: ids}

	newIDs := []openapi_types.UUID{uuid.New(), uuid.New()}
	task.ApplyUpdate(api.TaskUpdate{AssigneeIds: &newIDs}, false)

	if len(task.AssigneeIDs) != 1 || task.AssigneeIDs[0] != ids[0] {
		t.Errorf("AssigneeIDs = %v, want unchanged %v (ApplyUpdate must not touch assignees)", task.AssigneeIDs, ids)
	}
}

func TestTask_ToAPI_EmptyAssigneesNormalizeToEmptySliceNotNull(t *testing.T) {
	task := Task{ID: uuid.New(), ProjectID: uuid.New(), Title: "Unassigned work"}

	out := task.ToAPI()

	if out.AssigneeIds == nil {
		t.Error("AssigneeIds = nil, want a non-nil empty slice (API contract: [] means unassigned, never null)")
	}
	if len(out.AssigneeIds) != 0 {
		t.Errorf("AssigneeIds = %v, want empty", out.AssigneeIds)
	}
}

func TestTask_ToAPI_CarriesAllAssigneeIDs(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	task := Task{ID: uuid.New(), AssigneeIDs: []uuid.UUID{a, b}}

	out := task.ToAPI()

	if len(out.AssigneeIds) != 2 || out.AssigneeIds[0] != a || out.AssigneeIds[1] != b {
		t.Errorf("AssigneeIds = %v, want [%v %v]", out.AssigneeIds, a, b)
	}
}
