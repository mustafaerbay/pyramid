package handlers

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// The hand-authored test schema (testutil_test.go) declares no foreign-key
// constraints, so tests below reference project/person/milestone ids that
// were never inserted anywhere — exactly like the production schema's
// task_assignees table, which only cares that ids round-trip, not that they
// resolve against another table. This keeps each test focused on the task
// handler logic under test instead of fixture setup for unrelated tables.

func TestTasks_Create_AssignsSequentialNumberPerProject(t *testing.T) {
	r, _ := newTestRouter(t)
	projectA, projectB := uuid.New(), uuid.New()

	var first, second, third api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: projectA, Title: "First"}, &first)
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: projectA, Title: "Second"}, &second)
	// A task in a different project restarts numbering — it's per-project,
	// not global.
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: projectB, Title: "Third"}, &third)

	if first.Number != 1 {
		t.Errorf("first.Number = %d, want 1", first.Number)
	}
	if second.Number != 2 {
		t.Errorf("second.Number = %d, want 2", second.Number)
	}
	if third.Number != 1 {
		t.Errorf("third.Number = %d, want 1 (numbering is per-project)", third.Number)
	}
}

func TestTasks_Create_DefaultsStatusAndPriority(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Task
	status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Untriaged work",
	}, &created)

	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", status, http.StatusCreated)
	}
	if created.Status != api.TaskStatusBacklog {
		t.Errorf("Status = %q, want backlog (default)", created.Status)
	}
	if created.Priority != api.TaskPriorityMedium {
		t.Errorf("Priority = %q, want medium (default)", created.Priority)
	}
	if len(created.AssigneeIds) != 0 {
		t.Errorf("AssigneeIds = %v, want empty when omitted at creation", created.AssigneeIds)
	}
}

func TestTasks_Create_WithAssignees(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob := uuid.New(), uuid.New()

	var created api.Task
	status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Pair task", AssigneeIds: &[]uuid.UUID{alice, bob},
	}, &created)

	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", status, http.StatusCreated)
	}
	if len(created.AssigneeIds) != 2 {
		t.Fatalf("AssigneeIds = %v, want 2 entries", created.AssigneeIds)
	}

	// Re-fetching independently confirms attachAssignees loads them back on
	// GET too, not just echoed from the create response.
	var fetched api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+created.Id.String(), nil, &fetched)
	if len(fetched.AssigneeIds) != 2 {
		t.Errorf("fetched AssigneeIds = %v, want 2 entries", fetched.AssigneeIds)
	}
}

func TestTasks_Update_AssigneeIds_ReplaceWholeSet(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Task", AssigneeIds: &[]uuid.UUID{alice, bob},
	}, &created)

	// Present-and-non-empty: replaces the whole set.
	var replaced api.Task
	status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(),
		api.TaskUpdate{AssigneeIds: &[]uuid.UUID{carol}}, &replaced)
	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if len(replaced.AssigneeIds) != 1 || replaced.AssigneeIds[0] != carol {
		t.Errorf("AssigneeIds = %v, want [%v]", replaced.AssigneeIds, carol)
	}

	// Present-but-empty: unassigns everyone (distinct from "absent").
	var cleared api.Task
	status = doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(),
		api.TaskUpdate{AssigneeIds: &[]uuid.UUID{}}, &cleared)
	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if len(cleared.AssigneeIds) != 0 {
		t.Errorf("AssigneeIds = %v, want empty after clearing", cleared.AssigneeIds)
	}
}

func TestTasks_Update_AssigneeIds_AbsentLeavesUntouched(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob := uuid.New(), uuid.New()

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Task", AssigneeIds: &[]uuid.UUID{alice, bob},
	}, &created)

	// A patch that doesn't mention assigneeIds at all must leave the
	// existing assignee set alone — this is the "absent means untouched"
	// half of the *[]uuid.UUID contract in UpdateTask.
	title := "Renamed task"
	var updated api.Task
	status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(),
		map[string]any{"title": title}, &updated)

	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if updated.Title != title {
		t.Errorf("Title = %q, want %q", updated.Title, title)
	}
	if len(updated.AssigneeIds) != 2 {
		t.Errorf("AssigneeIds = %v, want unchanged 2 entries", updated.AssigneeIds)
	}
}

func TestTasks_Update_MilestoneId_ExplicitNullClearsIt(t *testing.T) {
	r, _ := newTestRouter(t)
	milestoneID := uuid.New()

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Task", MilestoneId: &milestoneID,
	}, &created)
	if created.MilestoneId == nil || *created.MilestoneId != milestoneID {
		t.Fatalf("created.MilestoneId = %v, want %v", created.MilestoneId, milestoneID)
	}

	// A raw JSON body with an explicit `"milestoneId": null` must clear it —
	// this is exactly the case isExplicitNull exists to detect, since a
	// *uuid.UUID field can't otherwise distinguish "sent null" from
	// "omitted" after json.Unmarshal.
	var cleared api.Task
	status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(),
		map[string]any{"milestoneId": nil}, &cleared)

	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if cleared.MilestoneId != nil {
		t.Errorf("MilestoneId = %v, want nil after explicit null", cleared.MilestoneId)
	}
}

func TestTasks_Update_MilestoneId_OmittedLeavesUntouched(t *testing.T) {
	r, _ := newTestRouter(t)
	milestoneID := uuid.New()

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Task", MilestoneId: &milestoneID,
	}, &created)

	// A patch that never mentions milestoneId must leave it exactly as-is —
	// the counterpart case to the explicit-null test above.
	var updated api.Task
	status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(),
		map[string]any{"title": "Still on the milestone"}, &updated)

	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if updated.MilestoneId == nil || *updated.MilestoneId != milestoneID {
		t.Errorf("MilestoneId = %v, want unchanged %v", updated.MilestoneId, milestoneID)
	}
}

func TestTasks_Update_MilestoneId_SetToNewValue(t *testing.T) {
	r, _ := newTestRouter(t)
	oldMilestone, newMilestone := uuid.New(), uuid.New()

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Task", MilestoneId: &oldMilestone,
	}, &created)

	var updated api.Task
	status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(),
		api.TaskUpdate{MilestoneId: &newMilestone}, &updated)

	if status != http.StatusOK {
		t.Fatalf("update status = %d, want %d", status, http.StatusOK)
	}
	if updated.MilestoneId == nil || *updated.MilestoneId != newMilestone {
		t.Errorf("MilestoneId = %v, want %v", updated.MilestoneId, newMilestone)
	}
}

func TestTasks_List_FilterByAssignee(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob := uuid.New(), uuid.New()
	projectID := uuid.New()

	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: projectID, Title: "Alice's task", AssigneeIds: &[]uuid.UUID{alice},
	}, nil)
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: projectID, Title: "Bob's task", AssigneeIds: &[]uuid.UUID{bob},
	}, nil)
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: projectID, Title: "Both", AssigneeIds: &[]uuid.UUID{alice, bob},
	}, nil)

	var list api.TaskList
	status := doJSON(t, r, http.MethodGet, "/api/v1/tasks?assigneeId="+alice.String(), nil, &list)
	if status != http.StatusOK {
		t.Fatalf("list status = %d, want %d", status, http.StatusOK)
	}
	if len(list.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2 (Alice's task + Both)", len(list.Data))
	}
	for _, task := range list.Data {
		found := false
		for _, a := range task.AssigneeIds {
			if a == alice {
				found = true
			}
		}
		if !found {
			t.Errorf("task %q has AssigneeIds %v, want to include Alice", task.Title, task.AssigneeIds)
		}
	}
}

func TestTasks_Delete(t *testing.T) {
	r, _ := newTestRouter(t)

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "Task"}, &created)

	status := doJSON(t, r, http.MethodDelete, "/api/v1/tasks/"+created.Id.String(), nil, nil)
	if status != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", status, http.StatusNoContent)
	}

	status = doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+created.Id.String(), nil, nil)
	if status != http.StatusNotFound {
		t.Fatalf("get-after-delete status = %d, want %d", status, http.StatusNotFound)
	}
}

func mdDate(s string) *openapi_types.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return &openapi_types.Date{Time: t}
}

func manDaysOf(task api.Task) map[uuid.UUID]float64 {
	out := map[uuid.UUID]float64{}
	for _, a := range task.Assignments {
		out[a.PersonId] = a.ManDays
	}
	return out
}

func TestTasks_Create_WithAssignmentsAndStartDate(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob := uuid.New(), uuid.New()

	var created api.Task
	status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Requirement",
		StartDate: mdDate("2026-10-26"), DueDate: mdDate("2026-11-06"),
		Assignments: &[]api.TaskAssignment{{PersonId: alice, ManDays: 30}, {PersonId: bob, ManDays: 10}},
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", status)
	}
	if created.ManDays != 40 {
		t.Errorf("ManDays = %v, want 40", created.ManDays)
	}
	if m := manDaysOf(created); m[alice] != 30 || m[bob] != 10 {
		t.Errorf("assignments = %v, want alice 30, bob 10", m)
	}
	if len(created.AssigneeIds) != 2 {
		t.Errorf("AssigneeIds = %v, want both people", created.AssigneeIds)
	}
	if created.StartDate == nil || created.StartDate.Format("2006-01-02") != "2026-10-26" {
		t.Errorf("StartDate = %v, want 2026-10-26", created.StartDate)
	}

	var fetched api.Task
	doJSON(t, r, http.MethodGet, "/api/v1/tasks/"+created.Id.String(), nil, &fetched)
	if fetched.ManDays != 40 || manDaysOf(fetched)[alice] != 30 {
		t.Errorf("GET lost man-days: total %v, assignments %v", fetched.ManDays, manDaysOf(fetched))
	}

	var list api.TaskList
	doJSON(t, r, http.MethodGet, "/api/v1/tasks?projectId="+created.ProjectId.String(), nil, &list)
	if len(list.Data) != 1 || list.Data[0].ManDays != 40 {
		t.Errorf("list man-days = %+v, want one task with 40", list.Data)
	}
}

func TestTasks_Update_AssignmentsReplaceSetAndWinOverAssigneeIds(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob, carol := uuid.New(), uuid.New(), uuid.New()
	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "T",
		Assignments: &[]api.TaskAssignment{{PersonId: alice, ManDays: 5}},
	}, &created)

	var updated api.Task
	status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(), api.TaskUpdate{
		AssigneeIds: &[]uuid.UUID{carol},
		Assignments: &[]api.TaskAssignment{{PersonId: bob, ManDays: 2.5}},
	}, &updated)
	if status != http.StatusOK {
		t.Fatalf("update status = %d, want 200", status)
	}
	if m := manDaysOf(updated); len(m) != 1 || m[bob] != 2.5 {
		t.Errorf("assignments = %v, want only bob 2.5", m)
	}
	if updated.ManDays != 2.5 {
		t.Errorf("ManDays = %v, want 2.5", updated.ManDays)
	}
}

func TestTasks_AssigneeIdsAloneMeansZeroManDays(t *testing.T) {
	r, _ := newTestRouter(t)
	alice := uuid.New()
	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "T", AssigneeIds: &[]uuid.UUID{alice},
	}, &created)
	if m := manDaysOf(created); len(m) != 1 || m[alice] != 0 || created.ManDays != 0 {
		t.Errorf("assignments = %v total %v, want alice with 0", m, created.ManDays)
	}
}

func TestTasks_RejectsBadManDaysAndDates(t *testing.T) {
	r, _ := newTestRouter(t)
	alice := uuid.New()
	cases := map[string]api.TaskCreate{
		"negative man-days": {ProjectId: uuid.New(), Title: "T", Assignments: &[]api.TaskAssignment{{PersonId: alice, ManDays: -1}}},
		"too many man-days": {ProjectId: uuid.New(), Title: "T", Assignments: &[]api.TaskAssignment{{PersonId: alice, ManDays: 1000}}},
		"start after due":   {ProjectId: uuid.New(), Title: "T", StartDate: mdDate("2026-11-10"), DueDate: mdDate("2026-11-01")},
	}
	for name, body := range cases {
		if status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", body, nil); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, status)
		}
	}

	var created api.Task
	doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "T", DueDate: mdDate("2026-11-01")}, &created)
	if status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(), api.TaskUpdate{StartDate: mdDate("2026-11-05")}, nil); status != http.StatusBadRequest {
		t.Errorf("patch start after existing due: status = %d, want 400", status)
	}
}

func TestTasks_EstimateManDays_UsedUntilAssignmentsCarryManDays(t *testing.T) {
	r, _ := newTestRouter(t)
	alice, bob := uuid.New(), uuid.New()
	est := 40.0

	var created api.Task
	status := doJSON(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{
		ProjectId: uuid.New(), Title: "Unassigned requirement", EstimateManDays: &est,
	}, &created)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d", status)
	}
	if created.EstimateManDays == nil || *created.EstimateManDays != 40 || created.ManDays != 40 {
		t.Errorf("estimate %v, manDays %v; want 40 and 40 (no assignments yet)", created.EstimateManDays, created.ManDays)
	}

	var split api.Task
	doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(), api.TaskUpdate{
		Assignments: &[]api.TaskAssignment{{PersonId: alice, ManDays: 30}, {PersonId: bob, ManDays: 15}},
	}, &split)
	if split.ManDays != 45 {
		t.Errorf("manDays = %v, want 45 (assignments win once they carry man-days)", split.ManDays)
	}

	bad := -1.0
	if status := doJSON(t, r, http.MethodPatch, "/api/v1/tasks/"+created.Id.String(), api.TaskUpdate{EstimateManDays: &bad}, nil); status != http.StatusBadRequest {
		t.Errorf("negative estimate: status = %d, want 400", status)
	}
}
