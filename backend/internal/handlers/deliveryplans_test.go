package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func TestDeliveryPlans(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")

	var p, other api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "DP", Name: "Plan"}, admin, &p)
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "OT", Name: "Other"}, admin, &other)
	var team map[string]any
	call(t, r, http.MethodPost, "/api/v1/project-teams", map[string]any{"projectId": p.Id, "name": "Web"}, admin, &team)
	teamID := uuid.MustParse(team["id"].(string))

	day := openapi_types.Date{}
	_ = day.UnmarshalText([]byte("2026-12-15"))
	in := api.DeliveryPlanInput{
		Name:    "Q1",
		Lanes:   []api.DeliveryPlanLane{{ProjectId: p.Id}, {ProjectId: p.Id, TeamId: &teamID}},
		Types:   []api.TaskType{api.TaskTypeEpic},
		Months:  api.DeliveryPlanInputMonthsN6,
		Markers: []api.DeliveryPlanMarker{{Date: day, Label: "Code freeze", Color: api.WorkItemTypeColorRed}},
	}
	var plan api.DeliveryPlan
	if s, _ := call(t, r, http.MethodPost, "/api/v1/delivery-plans", in, amy, &plan); s != http.StatusCreated {
		t.Fatalf("create = %d", s)
	}
	if !plan.Mine || len(plan.Lanes) != 2 || plan.Lanes[1].TeamId == nil || plan.Markers[0].Label != "Code freeze" || plan.Months != 6 {
		t.Errorf("plan = %+v", plan)
	}
	var list api.DeliveryPlanList
	call(t, r, http.MethodGet, "/api/v1/delivery-plans", nil, bob, &list)
	if len(list.Data) != 0 {
		t.Errorf("bob sees a private plan")
	}
	in.Shared = ptrTo(true)
	path := "/api/v1/delivery-plans/" + plan.Id.String()
	call(t, r, http.MethodPatch, path, in, amy, &plan)
	call(t, r, http.MethodGet, "/api/v1/delivery-plans", nil, bob, &list)
	if len(list.Data) != 1 || list.Data[0].Mine {
		t.Errorf("bob's list = %+v", list.Data)
	}
	if s, _ := call(t, r, http.MethodPatch, path, in, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob edits = %d", s)
	}

	bad := func(name string, mutate func(*api.DeliveryPlanInput)) {
		t.Helper()
		x := in
		x.Lanes = append([]api.DeliveryPlanLane{}, in.Lanes...)
		x.Markers = append([]api.DeliveryPlanMarker{}, in.Markers...)
		mutate(&x)
		if s, _ := call(t, r, http.MethodPost, "/api/v1/delivery-plans", x, amy, nil); s != http.StatusBadRequest {
			t.Errorf("%s = %d, want 400", name, s)
		}
	}
	bad("unknown project", func(x *api.DeliveryPlanInput) { x.Lanes = []api.DeliveryPlanLane{{ProjectId: uuid.New()}} })
	bad("team of another project", func(x *api.DeliveryPlanInput) {
		x.Lanes = []api.DeliveryPlanLane{{ProjectId: other.Id, TeamId: &teamID}}
	})
	bad("bad months", func(x *api.DeliveryPlanInput) { x.Months = 5 })
	bad("bad type", func(x *api.DeliveryPlanInput) { x.Types = []api.TaskType{"saga"} })
	bad("blank marker", func(x *api.DeliveryPlanInput) { x.Markers[0].Label = " " })
	bad("bad marker colour", func(x *api.DeliveryPlanInput) { x.Markers[0].Color = "plaid" })
	bad("blank name", func(x *api.DeliveryPlanInput) { x.Name = "" })

	if s, _ := call(t, r, http.MethodDelete, path, nil, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob deletes = %d", s)
	}
	if s, _ := call(t, r, http.MethodDelete, path, nil, amy, nil); s != http.StatusNoContent {
		t.Errorf("amy deletes = %d", s)
	}
}
