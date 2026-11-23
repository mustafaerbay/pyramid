package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestComments_WriteEditDelete(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "val@x.dev", api.SystemRoleViewer)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")
	val := login(t, r, "val@x.dev", "first-pass")

	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "T"}, amy, &task)
	base := "/api/v1/tasks/" + task.Id.String() + "/comments"

	var first, second api.TaskComment
	if s, _ := call(t, r, http.MethodPost, base, api.TaskCommentInput{Body: "  First!  "}, amy, &first); s != http.StatusCreated {
		t.Fatalf("amy comments: %d", s)
	}
	var amyMe api.AuthMe
	call(t, r, http.MethodGet, "/api/v1/auth/me", nil, amy, &amyMe)
	if first.Body != "First!" || first.AuthorPersonId == nil || *first.AuthorPersonId != amyMe.Person.Id {
		t.Errorf("comment = %+v, want trimmed body by amy", first)
	}
	call(t, r, http.MethodPost, base, api.TaskCommentInput{Body: "Second"}, bob, &second)

	if s, _ := call(t, r, http.MethodPost, base, api.TaskCommentInput{Body: "   "}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("blank comment = %d, want 400", s)
	}
	if s, _ := call(t, r, http.MethodPost, base, api.TaskCommentInput{Body: strings.Repeat("x", 5001)}, amy, nil); s != http.StatusBadRequest {
		t.Errorf("too long comment = %d, want 400", s)
	}
	if s, _ := call(t, r, http.MethodPost, base, api.TaskCommentInput{Body: "hi"}, val, nil); s != http.StatusForbidden {
		t.Errorf("viewer comment = %d, want 403", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/comments", api.TaskCommentInput{Body: "hi"}, amy, nil); s != http.StatusNotFound {
		t.Errorf("comment on missing task = %d, want 404", s)
	}

	var list api.TaskCommentList
	call(t, r, http.MethodGet, base, nil, val, &list)
	if len(list.Data) != 2 || list.Data[0].Id != first.Id || list.Data[1].Id != second.Id {
		t.Fatalf("list = %+v, want [first second]", list.Data)
	}

	edit := "/api/v1/task-comments/" + first.Id.String()
	if s, _ := call(t, r, http.MethodPatch, edit, api.TaskCommentInput{Body: "hijack"}, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob edits amy's comment = %d, want 403", s)
	}
	var edited api.TaskComment
	if s, _ := call(t, r, http.MethodPatch, edit, api.TaskCommentInput{Body: "First, edited"}, amy, &edited); s != http.StatusOK || edited.Body != "First, edited" {
		t.Errorf("amy edits = %d %q", s, edited.Body)
	}
	if s, _ := call(t, r, http.MethodDelete, edit, nil, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob deletes amy's comment = %d, want 403", s)
	}
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/task-comments/"+second.Id.String(), nil, bob, nil); s != http.StatusNoContent {
		t.Errorf("bob deletes own = %d, want 204", s)
	}
	if s, _ := call(t, r, http.MethodDelete, edit, nil, admin, nil); s != http.StatusNoContent {
		t.Errorf("admin deletes = %d, want 204", s)
	}
	call(t, r, http.MethodGet, base, nil, amy, &list)
	if len(list.Data) != 0 {
		t.Errorf("after deletes %d comments left", len(list.Data))
	}
}

func TestHistory_RecordsWhoChangedWhat(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	var me api.AuthMe
	call(t, r, http.MethodGet, "/api/v1/auth/me", nil, admin, &me)

	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "Old title", Tags: tagsPtr("a")}, admin, &task)
	path := "/api/v1/tasks/" + task.Id.String()
	call(t, r, http.MethodPatch, path, map[string]any{"title": "New title", "status": "in_progress", "tags": []string{"a", "b"}}, admin, nil)
	// A PATCH that changes nothing adds no entry.
	call(t, r, http.MethodPatch, path, map[string]any{"title": "New title"}, admin, nil)

	var h api.TaskHistory
	if s, _ := call(t, r, http.MethodGet, path+"/history", nil, admin, &h); s != http.StatusOK {
		t.Fatalf("history = %d", s)
	}
	if len(h.Data) != 2 {
		t.Fatalf("history has %d entries, want 2 (updated, created): %+v", len(h.Data), h.Data)
	}
	upd, created := h.Data[0], h.Data[1]
	if created.Action != "created" || upd.Action != "updated" {
		t.Errorf("actions = %s, %s; want updated, created (newest first)", upd.Action, created.Action)
	}
	if upd.ActorPersonId == nil || *upd.ActorPersonId != me.Person.Id {
		t.Errorf("actor = %v, want %s", upd.ActorPersonId, me.Person.Id)
	}
	if c := upd.Changes["title"]; c.From != "Old title" || c.To != "New title" {
		t.Errorf("title change = %+v", c)
	}
	if c := upd.Changes["status"]; c.From != "backlog" || c.To != "in_progress" {
		t.Errorf("status change = %+v", c)
	}
	if _, ok := upd.Changes["tags"]; !ok {
		t.Errorf("tags change missing: %+v", upd.Changes)
	}
	if _, ok := upd.Changes["priority"]; ok {
		t.Errorf("unchanged priority listed: %+v", upd.Changes)
	}
}
