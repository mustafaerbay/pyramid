package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func day(offset int) openapi_types.Date {
	return openapi_types.Date{Time: time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, offset)}
}

func TestSprints_CRUDAndTasks(t *testing.T) {
	r, _ := newTestRouter(t)
	project := uuid.New()
	var sp api.Sprint
	if s := doJSON(t, r, http.MethodPost, "/api/v1/sprints", api.SprintCreate{ProjectId: project, Name: "Sprint 1", StartDate: day(-2), EndDate: day(11)}, &sp); s != http.StatusCreated {
		t.Fatalf("create = %d", s)
	}
	if sp.Status != api.SprintStatusPlanned || sp.Stats.TaskCount != 0 {
		t.Errorf("new sprint = %+v", sp)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/sprints", api.SprintCreate{ProjectId: project, Name: "Bad", StartDate: day(5), EndDate: day(1)}, nil); s != http.StatusBadRequest {
		t.Errorf("end before start = %d, want 400", s)
	}

	a := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A", SprintId: &sp.Id, StoryPoints: f64(3), EstimateManDays: f64(2)})
	b := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "B", StoryPoints: f64(5)})
	newTask(t, r, api.TaskCreate{ProjectId: project, Title: "C"})
	if a.SprintId == nil || *a.SprintId != sp.Id {
		t.Fatalf("A sprint = %v", a.SprintId)
	}
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+b.Id.String(), map[string]any{"sprintId": sp.Id, "status": "done"}, nil)

	var in, none api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?sprintId="+sp.Id.String(), nil, &in)
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?projectId="+project.String()+"&sprintId=none", nil, &none)
	if len(in.Data) != 2 || len(none.Data) != 1 {
		t.Errorf("in sprint %d, in none %d; want 2 and 1", len(in.Data), len(none.Data))
	}

	var got api.Sprint
	doJSON(t, r, http.MethodGet, "/api/v1/sprints/"+sp.Id.String(), nil, &got)
	if st := got.Stats; st.TaskCount != 2 || st.DoneCount != 1 || st.Points != 8 || st.DonePoints != 5 || st.ManDays != 2 {
		t.Errorf("stats = %+v", st)
	}

	// A task can't go into another project's sprint.
	var other api.Sprint
	doJSON(t, r, http.MethodPost, "/api/v1/sprints", api.SprintCreate{ProjectId: uuid.New(), Name: "X", StartDate: day(0), EndDate: day(5)}, &other)
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), map[string]any{"sprintId": other.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("sprint of another project = %d, want 400", s)
	}

	status := api.SprintStatusActive
	doJSON(t, r, http.MethodPatch, "/api/v1/sprints/"+sp.Id.String(), api.SprintUpdate{Status: &status}, &got)
	if got.Status != api.SprintStatusActive {
		t.Errorf("status = %s", got.Status)
	}
	var list api.SprintList
	doJSON(t, r, http.MethodGet, "/api/v1/sprints?projectId="+project.String(), nil, &list)
	if len(list.Data) != 1 {
		t.Errorf("list = %d sprints", len(list.Data))
	}

	// Deleting the sprint sends its items back to the backlog.
	if s := doJSON(t, r, http.MethodDelete, "/api/v1/sprints/"+sp.Id.String(), nil, nil); s != http.StatusNoContent {
		t.Errorf("delete = %d", s)
	}
	var back api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+a.Id.String(), nil, &back)
	if back.SprintId != nil {
		t.Errorf("A sprint after delete = %v", back.SprintId)
	}
}

func TestSprints_Report(t *testing.T) {
	r, _ := newTestRouter(t)
	var me api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Me", LastName: "M", Email: "me@x.dev"}, &me)
	project := uuid.New()
	var sp api.Sprint
	doJSON(t, r, http.MethodPost, "/api/v1/sprints", api.SprintCreate{ProjectId: project, Name: "S", StartDate: day(-2), EndDate: day(9)}, &sp)
	a := newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A", SprintId: &sp.Id, StoryPoints: f64(3),
		Assignments: &[]api.TaskAssignment{{PersonId: me.Id, ManDays: 4}}})
	newTask(t, r, api.TaskCreate{ProjectId: project, Title: "B", SprintId: &sp.Id, StoryPoints: f64(5)})
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+a.Id.String(), map[string]any{"status": "done"}, nil)

	var rep api.SprintReport
	if s := doJSON(t, r, http.MethodGet, "/api/v1/sprints/"+sp.Id.String()+"/report", nil, &rep); s != http.StatusOK {
		t.Fatalf("report = %d", s)
	}
	if len(rep.Days) != 3 {
		t.Fatalf("days = %d, want 3 (start..today)", len(rep.Days))
	}
	first, today := rep.Days[0], rep.Days[2]
	if first.ScopeCount != 0 {
		t.Errorf("first day scope = %d, want 0 (items were added today)", first.ScopeCount)
	}
	if today.ScopeCount != 2 || today.DoneCount != 1 || today.ScopePoints != 8 || today.DonePoints != 3 {
		t.Errorf("today = %+v", today)
	}
	if len(rep.Capacity) != 1 || rep.Capacity[0].PersonId != me.Id || rep.Capacity[0].PlannedManDays != 4 || rep.Capacity[0].AvailableManDays <= 0 {
		t.Errorf("capacity = %+v", rep.Capacity)
	}
}
