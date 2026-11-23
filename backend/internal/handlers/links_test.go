package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestLinks_RelatedBothWaysAndDuplicate(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	a := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A"})
	b := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "B"})
	c := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "C", RelatedIds: idsPtr(a.Id, b.Id, a.Id)})

	if len(c.RelatedIds) != 2 || c.RelatedIds[0] != a.Id || c.RelatedIds[1] != b.Id {
		t.Fatalf("C related = %v, want [A B]", c.RelatedIds)
	}
	var got api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+a.Id.String(), nil, &got)
	if len(got.RelatedIds) != 1 || got.RelatedIds[0] != c.Id {
		t.Errorf("A related = %v, want [C] (links go both ways)", got.RelatedIds)
	}
	if a.RelatedIds == nil {
		t.Error("relatedIds must be [] not null")
	}

	// Removing the link from A's side removes it for C too.
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), map[string]any{"relatedIds": []uuid.UUID{}}, nil)
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+c.Id.String(), nil, &got)
	if len(got.RelatedIds) != 1 || got.RelatedIds[0] != b.Id {
		t.Errorf("C related after A cleared = %v, want [B]", got.RelatedIds)
	}

	// Duplicates.
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+b.Id.String(), map[string]any{"duplicateOfId": a.Id}, &got)
	if got.DuplicateOfId == nil || *got.DuplicateOfId != a.Id {
		t.Fatalf("B duplicateOf = %v", got.DuplicateOfId)
	}
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), map[string]any{"duplicateOfId": b.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("A duplicate of its own duplicate = %d, want 400", s)
	}
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+b.Id.String(), map[string]any{"duplicateOfId": nil}, &got)
	if got.DuplicateOfId != nil {
		t.Errorf("after null duplicateOf = %v", got.DuplicateOfId)
	}

	other := newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "Elsewhere"})
	for name, body := range map[string]map[string]any{
		"related self":        {"relatedIds": []uuid.UUID{a.Id}},
		"related elsewhere":   {"relatedIds": []uuid.UUID{other.Id}},
		"duplicate self":      {"duplicateOfId": a.Id},
		"duplicate elsewhere": {"duplicateOfId": other.Id},
	} {
		if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), body, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}

	// Deleting a task drops its links.
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+c.Id.String(), map[string]any{"relatedIds": []uuid.UUID{a.Id}}, nil)
	doJSON(t, r, http.MethodDelete, "/api/v1/tasks/"+c.Id.String(), nil, nil)
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+a.Id.String(), nil, &got)
	if len(got.RelatedIds) != 0 {
		t.Errorf("A related after C deleted = %v", got.RelatedIds)
	}
}
