package handlers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestTestPlans(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "TP", Name: "Tests"}, &p)
	story := newTask(t, r, api.TaskCreate{ProjectId: p.Id, Title: "Login story"})

	var plan api.TestPlan
	if s := doJSON(t, r, http.MethodPost, "/api/v1/test-plans", api.TestPlanInput{ProjectId: p.Id, Name: "Release 1"}, &plan); s != http.StatusCreated {
		t.Fatalf("plan = %d", s)
	}
	var suite api.TestSuite
	doJSON(t, r, http.MethodPost, "/api/v1/test-suites", api.TestSuiteInput{PlanId: plan.Id, Name: "Login"}, &suite)
	steps := []api.TestStep{{Action: "Open /login", Expected: "Form shows"}, {Action: "Submit wrong password", Expected: "Error shows"}}
	var c1, c2 api.TestCase
	if s := doJSON(t, r, http.MethodPost, "/api/v1/test-cases", api.TestCaseInput{SuiteId: suite.Id, Title: "Wrong password", Steps: &steps, TaskId: &story.Id}, &c1); s != http.StatusCreated || len(c1.Steps) != 2 {
		t.Fatalf("case = %d %+v", s, c1)
	}
	doJSON(t, r, http.MethodPost, "/api/v1/test-cases", api.TestCaseInput{SuiteId: suite.Id, Title: "Happy path"}, &c2)

	var run api.TestRunDetail
	if s := doJSON(t, r, http.MethodPost, "/api/v1/test-runs", map[string]any{"planId": plan.Id}, &run); s != http.StatusCreated || len(run.Results) != 2 || run.NotRun != 2 {
		t.Fatalf("run = %d %+v", s, run)
	}
	var res1 api.TestResult
	for _, x := range run.Results {
		if x.CaseId != nil && *x.CaseId == c1.Id {
			res1 = x
		}
	}
	var got api.TestResult
	doJSON(t, r, http.MethodPatch, "/api/v1/test-results/"+res1.Id.String(), map[string]any{"stepOutcomes": []string{"passed", "failed"}, "comment": "500 error"}, &got)
	if got.Outcome != api.TestOutcomeFailed || len(got.StepOutcomes) != 2 {
		t.Errorf("result = %+v", got)
	}
	var bug api.Task
	if s := doJSON(t, r, http.MethodPost, "/api/v1/test-results/"+res1.Id.String()+"/bug", nil, &bug); s != http.StatusCreated || bug.Type != api.TaskTypeBug ||
		bug.Description == nil || !strings.Contains(*bug.Description, "Submit wrong password") || !strings.Contains(*bug.Description, "500 error") {
		t.Fatalf("bug = %d %+v", s, bug)
	}
	if len(bug.RelatedIds) != 1 || bug.RelatedIds[0] != story.Id {
		t.Errorf("bug not related to the story: %+v", bug.RelatedIds)
	}
	doJSON(t, r, http.MethodGet, "/api/v1/test-runs/"+run.Id.String(), nil, &run)
	if run.Failed != 1 || run.NotRun != 1 || run.Results[0].BugTaskId == nil && run.Results[1].BugTaskId == nil {
		t.Errorf("run after = %+v", run)
	}
	// Editing a case later doesn't change the run's copy.
	doJSON(t, r, http.MethodPatch, "/api/v1/test-cases/"+c1.Id.String(), api.TestCaseInput{SuiteId: suite.Id, Title: "Renamed"}, nil)
	var detail api.TestPlanDetail
	doJSON(t, r, http.MethodGet, "/api/v1/test-plans/"+plan.Id.String(), nil, &detail)
	if len(detail.Suites) != 1 || len(detail.Suites[0].Cases) != 2 || len(detail.Runs) != 1 || detail.Failed != 1 || detail.CaseCount != 2 {
		t.Errorf("detail = %+v", detail)
	}
	var plans api.TestPlanList
	doJSON(t, r, http.MethodGet, "/api/v1/test-plans?projectId="+p.Id.String(), nil, &plans)
	if len(plans.Data) != 1 || plans.Data[0].Failed != 1 {
		t.Errorf("plans = %+v", plans.Data)
	}
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/test-results/"+res1.Id.String(), map[string]any{"outcome": "maybe"}, nil); s != http.StatusBadRequest {
		t.Errorf("bad outcome = %d", s)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/test-cases", api.TestCaseInput{SuiteId: suite.Id, Title: " "}, nil); s != http.StatusBadRequest {
		t.Errorf("blank case = %d", s)
	}
	if s := doJSON(t, r, http.MethodDelete, "/api/v1/test-plans/"+plan.Id.String(), nil, nil); s != http.StatusNoContent {
		t.Errorf("delete plan = %d", s)
	}
}
