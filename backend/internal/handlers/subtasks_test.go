package handlers

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func newTask(t *testing.T, r *gin.Engine, in api.TaskCreate) api.Task {
	t.Helper()
	var out api.Task
	if status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", in, &out); status != http.StatusCreated {
		t.Fatalf("create %q: status %d", in.Title, status)
	}
	return out
}

func TestSubtasks_CreateAndClear(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	parent := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Parent"})
	if parent.ParentId != nil {
		t.Errorf("top-level task ParentId = %v, want nil", parent.ParentId)
	}
	child := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Child", ParentId: &parent.Id})
	if child.ParentId == nil || *child.ParentId != parent.Id {
		t.Fatalf("child ParentId = %v, want %s", child.ParentId, parent.Id)
	}

	path := "/api/v1/tasks/" + child.Id.String()
	var got api.Task
	// Omitting parentId leaves it alone.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"title": "Child 2"}, &got)
	if got.ParentId == nil || *got.ParentId != parent.Id {
		t.Errorf("after title PATCH ParentId = %v, want parent kept", got.ParentId)
	}
	// Explicit null makes it top-level again.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"parentId": nil}, &got)
	if got.ParentId != nil {
		t.Errorf("after null PATCH ParentId = %v, want nil", got.ParentId)
	}
	// And it can be made a subtask again.
	doJSON(t, r, http.MethodPatch, path, map[string]any{"parentId": parent.Id}, &got)
	if got.ParentId == nil || *got.ParentId != parent.Id {
		t.Errorf("after set PATCH ParentId = %v, want parent", got.ParentId)
	}
}

func TestSubtasks_RejectsBadParents(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	parent := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Parent"})
	child := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Child", ParentId: &parent.Id})
	elsewhere := newTask(t, r, api.TaskCreate{ProjectId: uuid.New(), Title: "Other project"})
	loner := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Loner"})
	missing := uuid.New()

	creates := map[string]uuid.UUID{
		"missing parent":          missing,
		"parent in other project": elsewhere.Id,
	}
	for name, pid := range creates {
		status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: name, ParentId: &pid}, nil)
		if status != http.StatusBadRequest {
			t.Errorf("create with %s: status %d, want 400", name, status)
		}
	}

	patches := []struct {
		name   string
		task   uuid.UUID
		parent uuid.UUID
	}{
		{"own parent", loner.Id, loner.Id},
		{"under its own child (loop)", parent.Id, child.Id},
	}
	for _, p := range patches {
		status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+p.task.String(), map[string]any{"parentId": p.parent}, nil)
		if status != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", p.name, status)
		}
	}
}

func TestHierarchy_LevelsAndDepth(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	typ := func(v api.TaskType) *api.TaskType { return &v }

	epic := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Epic", Type: typ(api.TaskTypeEpic)})
	feature := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Feature", Type: typ(api.TaskTypeFeature), ParentId: &epic.Id})
	story := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Story", Type: typ(api.TaskTypeStory), ParentId: &feature.Id})
	bug := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Bug", Type: typ(api.TaskTypeBug), ParentId: &feature.Id})
	task := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Task", ParentId: &story.Id})
	if task.Type != api.TaskTypeTask || epic.Type != api.TaskTypeEpic {
		t.Errorf("types = %s, %s", task.Type, epic.Type)
	}
	// Same level nests (task under task), four deep is fine.
	sub := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Sub", ParentId: &task.Id})
	_ = bug

	// A higher level can't go under a lower one.
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "F", Type: typ(api.TaskTypeFeature), ParentId: &story.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("feature under story = %d, want 400", s)
	}
	// Changing a type so it outranks its parent is refused too.
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+story.Id.String(), map[string]any{"type": "epic"}, nil); s != http.StatusBadRequest {
		t.Errorf("story → epic under a feature = %d, want 400", s)
	}
	// …as is making a parent lower than its children.
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+feature.Id.String(), map[string]any{"type": "task"}, nil); s != http.StatusBadRequest {
		t.Errorf("feature → task above stories = %d, want 400", s)
	}
	// Depth: epic>feature>story>task>sub is 5 levels; a 6th is refused.
	if s := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: project, Title: "Too deep", ParentId: &sub.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("6th level = %d, want 400", s)
	}
	// Moving a whole branch must keep the depth too: the story branch is
	// 3 deep (story>task>sub), under sub's level it would be too deep.
	other := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "Other story", Type: typ(api.TaskTypeStory), ParentId: &feature.Id})
	o2 := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "o2", ParentId: &other.Id})
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+task.Id.String(), map[string]any{"parentId": o2.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("branch moved too deep = %d, want 400", s)
	}
}

// A task with subtasks is only a container: its own man-days don't count,
// so the same work isn't planned twice.
func TestSubtasks_ParentManDaysNotCountedTwice(t *testing.T) {
	r, _ := newTestRouter(t)
	var me api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Me", LastName: "M", Email: "me@x.dev"}, &me)
	var project api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "A", Name: "A"}, &project)

	due := mdDate("2026-12-10")
	parent := newTask(t, r, api.TaskCreate{ProjectId: project.Id, Title: "Epic", DueDate: due,
		Assignments: &[]api.TaskAssignment{{PersonId: me.Id, ManDays: 10}}})
	newTask(t, r, api.TaskCreate{ProjectId: project.Id, Title: "Part", DueDate: due, ParentId: &parent.Id,
		Assignments: &[]api.TaskAssignment{{PersonId: me.Id, ManDays: 4}}})

	var rep api.CapacityReport
	doJSON(t, r, http.MethodGet, "/api/v1/capacity?month=2026-12", nil, &rep)
	if rep.PlannedManDays != 4 {
		t.Errorf("capacity planned = %v, want 4 (only the subtask)", rep.PlannedManDays)
	}

	var wl api.PersonWorkload
	doJSON(t, r, http.MethodGet, "/api/v1/people/"+me.Id.String()+"/workload?year=2026", nil, &wl)
	for _, m := range wl.Months {
		if m.Month == "2026-12" {
			if m.PlannedManDays != 4 || len(m.Tasks) != 1 || m.Tasks[0].Title != "Part" {
				t.Errorf("workload Dec = %v md, tasks %v; want 4 md from Part only", m.PlannedManDays, m.Tasks)
			}
		}
	}

	// Once the subtask is gone, the parent counts again.
	var list api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?projectId="+project.Id.String(), nil, &list)
	for _, task := range list.Data {
		if task.ParentId != nil {
			doJSON(t, r, http.MethodDelete, "/api/v1/tasks/"+task.Id.String(), nil, nil)
		}
	}
	doJSON(t, r, http.MethodGet, "/api/v1/capacity?month=2026-12", nil, &rep)
	if rep.PlannedManDays != 10 {
		t.Errorf("after deleting subtask planned = %v, want 10", rep.PlannedManDays)
	}
}
