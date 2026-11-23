package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func idsPtr(ids ...uuid.UUID) *[]uuid.UUID {
	if ids == nil {
		ids = []uuid.UUID{}
	}
	return &ids
}

func TestDependencies_SetReadAndClear(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	a := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A"})
	b := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "B"})
	c := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "C", BlockedByIds: idsPtr(b.Id, a.Id, a.Id)})

	if len(a.BlockedByIds) != 0 || a.BlockedByIds == nil {
		t.Errorf("A BlockedByIds = %#v, want empty list", a.BlockedByIds)
	}
	// Deduplicated and sorted by task number.
	if len(c.BlockedByIds) != 2 || c.BlockedByIds[0] != a.Id || c.BlockedByIds[1] != b.Id {
		t.Fatalf("C BlockedByIds = %v, want [A B]", c.BlockedByIds)
	}

	var list api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?projectId="+project.String(), nil, &list)
	for _, task := range list.Data {
		if task.Id == c.Id && len(task.BlockedByIds) != 2 {
			t.Errorf("listed C BlockedByIds = %v, want 2", task.BlockedByIds)
		}
	}

	path := "/api/v1/tasks/" + c.Id.String()
	var got api.Task
	doJSON(t, r, http.MethodPatch, path, map[string]any{"title": "C2"}, &got)
	if len(got.BlockedByIds) != 2 {
		t.Errorf("after title PATCH BlockedByIds = %v, want kept", got.BlockedByIds)
	}
	doJSON(t, r, http.MethodPatch, path, map[string]any{"blockedByIds": []uuid.UUID{b.Id}}, &got)
	if len(got.BlockedByIds) != 1 || got.BlockedByIds[0] != b.Id {
		t.Errorf("after replace BlockedByIds = %v, want [B]", got.BlockedByIds)
	}
	doJSON(t, r, http.MethodPatch, path, map[string]any{"blockedByIds": []uuid.UUID{}}, &got)
	if len(got.BlockedByIds) != 0 {
		t.Errorf("after clear BlockedByIds = %v, want none", got.BlockedByIds)
	}
}

func TestDependencies_RejectsSelfOtherProjectAndLoops(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	a := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A"})
	b := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "B", BlockedByIds: idsPtr(a.Id)})
	c := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "C", BlockedByIds: idsPtr(b.Id)})
	other := newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "Elsewhere"})

	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "X", BlockedByIds: idsPtr(other.Id)}, nil); s != http.StatusBadRequest {
		t.Errorf("blocker in other project: status %d, want 400", s)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "Y", BlockedByIds: idsPtr(uuid.New())}, nil); s != http.StatusBadRequest {
		t.Errorf("missing blocker: status %d, want 400", s)
	}
	cases := map[string]struct {
		task    uuid.UUID
		blocker uuid.UUID
	}{
		"self":          {a.Id, a.Id},
		"direct loop":   {a.Id, b.Id},
		"indirect loop": {a.Id, c.Id},
	}
	for name, cs := range cases {
		s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+cs.task.String(), map[string]any{"blockedByIds": []uuid.UUID{cs.blocker}}, nil)
		if s != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", name, s)
		}
	}
}

func TestDependencies_GoneWhenBlockerDeleted(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	a := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A"})
	b := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "B", BlockedByIds: idsPtr(a.Id)})
	doJSON(t, r, http.MethodDelete, "/api/v1/tasks/"+a.Id.String(), nil, nil)
	var got api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+b.Id.String(), nil, &got)
	if len(got.BlockedByIds) != 0 {
		t.Errorf("BlockedByIds = %v after blocker deleted, want none", got.BlockedByIds)
	}
}
