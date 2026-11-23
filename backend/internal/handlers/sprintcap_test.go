package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestSprintCapacitySettings(t *testing.T) {
	r, _ := newTestRouter(t)
	var me api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Me", LastName: "M", Email: "me@x.dev"}, &me)
	project := uuid.New()
	var sp api.Sprint
	// 2026-12-07 (Mon) to 2026-12-18 (Fri): 10 working days.
	doJSON(t, r, http.MethodPost, "/api/v1/sprints", api.SprintCreate{ProjectId: project, Name: "S", StartDate: *mdDate("2026-12-07"), EndDate: *mdDate("2026-12-18")}, &sp)
	newTask(t, r, api.TaskCreate{ProjectId: project, Title: "A", SprintId: &sp.Id, Assignments: &[]api.TaskAssignment{{PersonId: me.Id, ManDays: 4}}})

	path := "/api/v1/sprints/" + sp.Id.String()
	var rep api.SprintReport
	doJSON(t, r, http.MethodGet, path+"/report", nil, &rep)
	before := rep.Capacity[0].AvailableManDays

	var got api.SprintReport
	body := map[string]any{"data": []map[string]any{{"personId": me.Id, "daysOff": 2, "perDay": 0.5, "activity": "testing"}}}
	if s := doJSON(t, r, http.MethodPut, path+"/capacity", body, &got); s != http.StatusOK {
		t.Fatalf("set = %d", s)
	}
	c := got.Capacity[0]
	if c.DaysOff != 2 || c.PerDay == nil || *c.PerDay != 0.5 || c.Activity == nil || *c.Activity != api.SprintActivityTesting {
		t.Errorf("settings = %+v", c)
	}
	if c.AvailableManDays != 4 { // (10 - 2) days × 0.5
		t.Errorf("available = %v, want 4 (was %v)", c.AvailableManDays, before)
	}
	// Days off alone scale the default.
	doJSON(t, r, http.MethodPut, path+"/capacity", map[string]any{"data": []map[string]any{{"personId": me.Id, "daysOff": 5}}}, &got)
	if a := got.Capacity[0].AvailableManDays; a <= 0 || a >= before {
		t.Errorf("with 5 days off = %v, before %v", a, before)
	}
	for name, bad := range map[string]map[string]any{
		"days off": {"personId": me.Id, "daysOff": 99},
		"per day":  {"personId": me.Id, "perDay": 3},
		"activity": {"personId": me.Id, "activity": "napping"},
	} {
		if s := doJSON(t, r, http.MethodPut, path+"/capacity", map[string]any{"data": []map[string]any{bad}}, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
}
