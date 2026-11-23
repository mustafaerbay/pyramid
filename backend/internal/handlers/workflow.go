package handlers

import (
	"fmt"
	"time"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

func (h *Handler) projectStates(t *models.Task) []api.WorkflowState {
	var p models.Project
	if h.db.Select("id, workflow").First(&p, "id = ?", t.ProjectID).Error != nil {
		return models.DefaultWorkflow()
	}
	return p.States()
}

// resolveState keeps State and Status in step. A requested state sets the
// status to its category; otherwise, when the status may have changed, the
// item keeps its state if it's still in that category or takes the
// category's first state.
func (h *Handler) resolveState(t *models.Task, want *string, statusChanged bool) error {
	states := h.projectStates(t)
	if want != nil {
		for _, s := range states {
			if s.Key == *want {
				t.State, t.Status = s.Key, s.Category
				h.settleDone(t)
				return nil
			}
		}
		return fmt.Errorf("this project has no state %q", *want)
	}
	if !statusChanged && t.State != "" {
		return nil
	}
	for _, s := range states {
		if s.Key == t.State && s.Category == t.Status {
			return nil
		}
	}
	for _, s := range states {
		if s.Category == t.Status {
			t.State = s.Key
			return nil
		}
	}
	return fmt.Errorf("this project's workflow has no %s state", t.Status)
}

// settleDone stamps completion when a state change makes the item done.
func (h *Handler) settleDone(t *models.Task) {
	if t.Status != api.TaskStatusDone {
		return
	}
	if t.CompletedAt == nil {
		now := time.Now().UTC()
		t.CompletedAt = &now
	}
	if t.RemainingManDays != nil {
		zero := 0.0
		t.RemainingManDays = &zero
	}
}
