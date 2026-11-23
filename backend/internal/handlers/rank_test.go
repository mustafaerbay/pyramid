package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func order(t *testing.T, r *gin.Engine, project uuid.UUID) []string {
	t.Helper()
	var l api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?pageSize=100&projectId="+project.String(), nil, &l)
	out := make([]string, len(l.Data))
	for i, x := range l.Data {
		out[i] = x.Title
	}
	return out
}

func move(t *testing.T, r *gin.Engine, id uuid.UUID, after *uuid.UUID) int {
	t.Helper()
	return doJSON(t, r, http.MethodPost, "/api/v1/tasks/"+id.String()+"/move", map[string]any{"afterId": after}, nil)
}

func TestRank_NewGoLastAndMovesStick(t *testing.T) {
	r, _ := newTestRouter(t)
	p := uuid.New()
	a := newTask(t, r, api.TaskCreate{ProjectId: p, Title: "A"})
	b := newTask(t, r, api.TaskCreate{ProjectId: p, Title: "B"})
	c := newTask(t, r, api.TaskCreate{ProjectId: p, Title: "C"})
	if got := order(t, r, p); got[0] != "A" || got[2] != "C" {
		t.Fatalf("initial order = %v", got)
	}
	if !(a.Rank < b.Rank && b.Rank < c.Rank) {
		t.Errorf("ranks = %v %v %v, want increasing", a.Rank, b.Rank, c.Rank)
	}

	move(t, r, c.Id, nil) // to the top
	if got := order(t, r, p); got[0] != "C" || got[1] != "A" || got[2] != "B" {
		t.Errorf("after C to top = %v", got)
	}
	move(t, r, c.Id, &a.Id) // right after A
	if got := order(t, r, p); got[0] != "A" || got[1] != "C" || got[2] != "B" {
		t.Errorf("after C after A = %v", got)
	}
	move(t, r, a.Id, &b.Id) // to the end
	if got := order(t, r, p); got[0] != "C" || got[1] != "B" || got[2] != "A" {
		t.Errorf("after A after B = %v", got)
	}
	// Many moves into the same gap keep working (ranks get spread again).
	for i := 0; i < 60; i++ {
		if i%2 == 0 {
			move(t, r, a.Id, &c.Id)
		} else {
			move(t, r, b.Id, &c.Id)
		}
	}
	if got := order(t, r, p); got[0] != "C" || got[1] != "B" || got[2] != "A" {
		t.Errorf("after many moves = %v, want [C B A]", got)
	}

	other := newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "X"})
	if s := move(t, r, a.Id, &other.Id); s != http.StatusBadRequest {
		t.Errorf("after a task in another project = %d, want 400", s)
	}
	if s := move(t, r, a.Id, &a.Id); s != http.StatusBadRequest {
		t.Errorf("after itself = %d, want 400", s)
	}
	if s := move(t, r, uuid.New(), nil); s != http.StatusNotFound {
		t.Errorf("missing task = %d, want 404", s)
	}
}
