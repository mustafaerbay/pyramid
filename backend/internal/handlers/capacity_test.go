package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

func f64(v float64) *float64 { return &v }

func TestPeople_MonthlyCapacity_DefaultsAndUpdates(t *testing.T) {
	r, _ := newTestRouter(t)

	var def, custom api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "D", LastName: "D", Email: "d@x.dev"}, &def)
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "C", LastName: "C", Email: "c@x.dev", MonthlyCapacityManDays: f64(22)}, &custom)
	if def.MonthlyCapacityManDays == nil || *def.MonthlyCapacityManDays != 20 {
		t.Errorf("default capacity = %v, want 20", def.MonthlyCapacityManDays)
	}
	if custom.MonthlyCapacityManDays == nil || *custom.MonthlyCapacityManDays != 22 {
		t.Errorf("custom capacity = %v, want 22", custom.MonthlyCapacityManDays)
	}

	var updated api.Person
	doJSON(t, r, http.MethodPatch, "/api/v1/people/"+def.Id.String(), api.PersonUpdate{MonthlyCapacityManDays: f64(18.5)}, &updated)
	if updated.MonthlyCapacityManDays == nil || *updated.MonthlyCapacityManDays != 18.5 {
		t.Errorf("updated capacity = %v, want 18.5", updated.MonthlyCapacityManDays)
	}

	if status := doJSON(t, r, http.MethodPatch, "/api/v1/people/"+def.Id.String(), api.PersonUpdate{MonthlyCapacityManDays: f64(32)}, nil); status != http.StatusBadRequest {
		t.Errorf("capacity 32: status = %d, want 400", status)
	}
	if status := doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "N", LastName: "N", Email: "n@x.dev", MonthlyCapacityManDays: f64(-1)}, nil); status != http.StatusBadRequest {
		t.Errorf("capacity -1: status = %d, want 400", status)
	}
}

func TestPeople_WorkloadPct_IsThisMonthsManDaysOverCapacity(t *testing.T) {
	r, _ := newTestRouter(t)

	var busy, idle api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Busy", LastName: "B", Email: "busy@x.dev", MonthlyCapacityManDays: f64(10)}, &busy)
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Idle", LastName: "I", Email: "idle@x.dev"}, &idle)
	var project api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "A", Name: "A"}, &project)

	today := openapi_types.Date{Time: time.Now().UTC()}
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: project.Id, Title: "Due today", DueDate: &today,
		Assignments: &[]api.TaskAssignment{{PersonId: busy.Id, ManDays: 5}},
	}, nil)

	var list api.PersonList
	if status := doJSON(t, r, http.MethodGet, "/api/v1/people", nil, &list); status != http.StatusOK {
		t.Fatalf("list status = %d", status)
	}
	for _, p := range list.Data {
		if p.WorkloadPct == nil || p.MonthManDays == nil {
			t.Fatalf("%s: workloadPct/monthManDays missing", p.FirstName)
		}
		switch p.Id {
		case busy.Id:
			if *p.MonthManDays != 5 || *p.WorkloadPct != 50 {
				t.Errorf("busy: %v man-days, %v%%; want 5, 50", *p.MonthManDays, *p.WorkloadPct)
			}
		case idle.Id:
			if *p.MonthManDays != 0 || *p.WorkloadPct != 0 {
				t.Errorf("idle: %v man-days, %v%%; want 0, 0", *p.MonthManDays, *p.WorkloadPct)
			}
		}
	}

	var one api.Person
	doJSON(t, r, http.MethodGet, "/api/v1/people/"+busy.Id.String(), nil, &one)
	if one.WorkloadPct == nil || *one.WorkloadPct != 50 {
		t.Errorf("GET workloadPct = %v, want 50", one.WorkloadPct)
	}
}

func TestCapacity_MonthReport(t *testing.T) {
	r, db := newTestRouter(t)

	var a, b, gone api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "A", LastName: "A", Email: "a@x.dev"}, &a)
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "B", LastName: "B", Email: "b@x.dev", MonthlyCapacityManDays: f64(22)}, &b)
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "G", LastName: "G", Email: "g@x.dev"}, &gone)
	inactive := api.PersonStatusInactive
	doJSON(t, r, http.MethodPatch, "/api/v1/people/"+gone.Id.String(), api.PersonUpdate{Status: &inactive}, nil)

	var live, dead api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "L", Name: "Live"}, &live)
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "D", Name: "Dead"}, &dead)
	if err := db.Exec("UPDATE projects SET status = 'cancelled' WHERE id = ?", dead.Id.String()).Error; err != nil {
		t.Fatal(err)
	}

	mk := func(project uuid.UUID, start, due *openapi_types.Date, status api.TaskStatus, as ...api.TaskAssignment) {
		t.Helper()
		if code := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
			ProjectId: project, Title: "t", StartDate: start, DueDate: due, Status: &status, Assignments: &as,
		}, nil); code != http.StatusCreated {
			t.Fatalf("create task: %d", code)
		}
	}
	// 2026-10-26..2026-11-06: half of the weekdays fall in November.
	mk(live.Id, mdDate("2026-10-26"), mdDate("2026-11-06"), api.TaskStatusInProgress,
		api.TaskAssignment{PersonId: a.Id, ManDays: 30}, api.TaskAssignment{PersonId: b.Id, ManDays: 10})
	mk(live.Id, nil, mdDate("2026-11-20"), api.TaskStatusDone, api.TaskAssignment{PersonId: b.Id, ManDays: 3})       // done still counts
	mk(live.Id, nil, mdDate("2026-11-20"), api.TaskStatusCancelled, api.TaskAssignment{PersonId: a.Id, ManDays: 10}) // cancelled task: ignored
	mk(dead.Id, nil, mdDate("2026-11-20"), api.TaskStatusTodo, api.TaskAssignment{PersonId: a.Id, ManDays: 5})       // cancelled project: ignored
	mk(live.Id, nil, nil, api.TaskStatusTodo, api.TaskAssignment{PersonId: b.Id, ManDays: 4})                        // undated: ignored
	mk(live.Id, nil, mdDate("2026-12-01"), api.TaskStatusTodo, api.TaskAssignment{PersonId: a.Id, ManDays: 7})       // other month

	var rep api.CapacityReport
	if status := doJSON(t, r, http.MethodGet, "/api/v1/capacity?month=2026-11", nil, &rep); status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if rep.Month != "2026-11" || rep.WorkingDays != 21 {
		t.Errorf("month %q, working days %d; want 2026-11, 21", rep.Month, rep.WorkingDays)
	}
	if rep.CapacityManDays != 42 {
		t.Errorf("capacity = %v, want 42 (inactive person excluded)", rep.CapacityManDays)
	}
	if rep.PlannedManDays != 23 {
		t.Errorf("planned = %v, want 23 (15 + 5 + 3)", rep.PlannedManDays)
	}
	people := map[uuid.UUID]api.PersonCapacity{}
	for _, p := range rep.People {
		people[p.PersonId] = p
	}
	if len(people) != 2 {
		t.Errorf("people = %+v, want only the 2 active people", rep.People)
	}
	if p := people[a.Id]; p.CapacityManDays != 20 || p.PlannedManDays != 15 {
		t.Errorf("A = %+v, want capacity 20, planned 15", p)
	}
	if p := people[b.Id]; p.CapacityManDays != 22 || p.PlannedManDays != 8 {
		t.Errorf("B = %+v, want capacity 22, planned 8", p)
	}
	if len(rep.Projects) != 1 || rep.Projects[0].ProjectId != live.Id || rep.Projects[0].PlannedManDays != 23 {
		t.Errorf("projects = %+v, want only Live with 23", rep.Projects)
	}
}

func TestCapacity_BadMonthIs400(t *testing.T) {
	r, _ := newTestRouter(t)
	for _, m := range []string{"2026-13", "nov", "2026-1"} {
		if status := doJSON(t, r, http.MethodGet, "/api/v1/capacity?month="+m, nil, nil); status != http.StatusBadRequest {
			t.Errorf("month=%s: status = %d, want 400", m, status)
		}
	}
}

func TestCapacity_DefaultsToCurrentMonth(t *testing.T) {
	r, _ := newTestRouter(t)
	var rep api.CapacityReport
	doJSON(t, r, http.MethodGet, "/api/v1/capacity", nil, &rep)
	if want := time.Now().UTC().Format("2006-01"); rep.Month != want {
		t.Errorf("month = %q, want %q", rep.Month, want)
	}
}

func TestPersonWorkload_YearByMonth(t *testing.T) {
	r, db := newTestRouter(t)

	var me, other api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "Me", LastName: "M", Email: "me@x.dev", MonthlyCapacityManDays: f64(18)}, &me)
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "O", LastName: "O", Email: "o@x.dev"}, &other)
	var live, dead api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "L", Name: "Live"}, &live)
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "D", Name: "Dead"}, &dead)
	db.Exec("UPDATE projects SET status = 'cancelled' WHERE id = ?", dead.Id.String())

	mk := func(project uuid.UUID, title string, start, due *openapi_types.Date, status api.TaskStatus, as ...api.TaskAssignment) api.Task {
		t.Helper()
		var task api.Task
		doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
			ProjectId: project, Title: title, StartDate: start, DueDate: due, Status: &status, Assignments: &as,
		}, &task)
		return task
	}
	split := mk(live.Id, "Split", mdDate("2026-10-26"), mdDate("2026-11-06"), api.TaskStatusInProgress,
		api.TaskAssignment{PersonId: me.Id, ManDays: 30}, api.TaskAssignment{PersonId: other.Id, ManDays: 10})
	mk(live.Id, "Cancelled", nil, mdDate("2026-11-20"), api.TaskStatusCancelled, api.TaskAssignment{PersonId: me.Id, ManDays: 9})
	mk(dead.Id, "Dead project", nil, mdDate("2026-11-20"), api.TaskStatusTodo, api.TaskAssignment{PersonId: me.Id, ManDays: 9})
	mk(live.Id, "Next year", nil, mdDate("2027-01-10"), api.TaskStatusTodo, api.TaskAssignment{PersonId: me.Id, ManDays: 4})

	var wl api.PersonWorkload
	if status := doJSON(t, r, http.MethodGet, "/api/v1/people/"+me.Id.String()+"/workload?year=2026", nil, &wl); status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if wl.Year != 2026 || wl.CapacityManDays != 18 || len(wl.Months) != 12 {
		t.Fatalf("year %d capacity %v months %d; want 2026, 18, 12", wl.Year, wl.CapacityManDays, len(wl.Months))
	}
	oct, nov, dec := wl.Months[9], wl.Months[10], wl.Months[11]
	if oct.Month != "2026-10" || oct.PlannedManDays != 15 || nov.PlannedManDays != 15 || dec.PlannedManDays != 0 {
		t.Errorf("oct %s %v, nov %v, dec %v; want 2026-10 15, 15, 0", oct.Month, oct.PlannedManDays, nov.PlannedManDays, dec.PlannedManDays)
	}
	if nov.WorkingDays != 21 {
		t.Errorf("nov working days = %d, want 21", nov.WorkingDays)
	}
	if len(nov.Projects) != 1 || nov.Projects[0].ProjectId != live.Id || nov.Projects[0].PlannedManDays != 15 {
		t.Errorf("nov projects = %+v, want Live 15", nov.Projects)
	}
	if len(nov.Tasks) != 1 {
		t.Fatalf("nov tasks = %+v, want only the split task", nov.Tasks)
	}
	got := nov.Tasks[0]
	if got.TaskId != split.Id || got.Title != "Split" || got.ManDays != 30 || got.MonthManDays != 15 || got.StartDate == nil {
		t.Errorf("nov task = %+v, want Split, 30 total, 15 this month, with start date", got)
	}

	var next api.PersonWorkload
	doJSON(t, r, http.MethodGet, "/api/v1/people/"+me.Id.String()+"/workload?year=2027", nil, &next)
	if next.Months[0].PlannedManDays != 4 {
		t.Errorf("2027-01 = %v, want 4", next.Months[0].PlannedManDays)
	}
}

func TestPersonWorkload_NotFoundAndBadYear(t *testing.T) {
	r, _ := newTestRouter(t)
	if status := doJSON(t, r, http.MethodGet, "/api/v1/people/"+uuid.New().String()+"/workload", nil, nil); status != http.StatusNotFound {
		t.Errorf("unknown person: %d, want 404", status)
	}
	var p api.Person
	doJSON(t, r, http.MethodPost, "/api/v1/people", api.PersonCreate{FirstName: "A", LastName: "A", Email: "a@x.dev"}, &p)
	for _, y := range []string{"abc", "1999", "2101"} {
		if status := doJSON(t, r, http.MethodGet, "/api/v1/people/"+p.Id.String()+"/workload?year="+y, nil, nil); status != http.StatusBadRequest {
			t.Errorf("year=%s: %d, want 400", y, status)
		}
	}
}
