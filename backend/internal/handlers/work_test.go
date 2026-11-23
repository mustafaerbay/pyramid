package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestWork_PointsRemainingCompleted(t *testing.T) {
	r, _ := newTestRouter(t)
	task := newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "T",
		StoryPoints: f64(5), RemainingManDays: f64(3), CompletedManDays: f64(1)})
	if task.StoryPoints == nil || *task.StoryPoints != 5 || *task.RemainingManDays != 3 || *task.CompletedManDays != 1 {
		t.Fatalf("created = %+v", task)
	}
	path := "/api/v1/tasks/" + task.Id.String()
	var got api.Task
	// Leaving them out keeps them; null clears.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"title": "T2"}, &got)
	if got.StoryPoints == nil || *got.StoryPoints != 5 {
		t.Errorf("points after title PATCH = %v", got.StoryPoints)
	}
	doJSON(t, r, http.MethodPatch, path, map[string]any{"storyPoints": nil, "completedManDays": 2.5}, &got)
	if got.StoryPoints != nil || got.CompletedManDays == nil || *got.CompletedManDays != 2.5 {
		t.Errorf("after clear/set = points %v completed %v", got.StoryPoints, got.CompletedManDays)
	}
	// Done clears the remaining work.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"status": "done"}, &got)
	if got.RemainingManDays == nil || *got.RemainingManDays != 0 {
		t.Errorf("remaining after done = %v, want 0", got.RemainingManDays)
	}
	for name, body := range map[string]map[string]any{
		"points":    {"storyPoints": 1000},
		"remaining": {"remainingManDays": -1},
		"completed": {"completedManDays": 10000},
	} {
		if s := doJSON(t, r, http.MethodPatch, path, body, nil); s != http.StatusBadRequest {
			t.Errorf("%s out of range = %d, want 400", name, s)
		}
	}
}
