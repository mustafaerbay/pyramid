package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Test plans: a plan holds suites of test cases (steps: action and
// expected result). A run copies the cases it covers, so later edits don't
// rewrite history, and records an outcome per step and per case.

type testPlan struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID  `gorm:"type:uuid;not null"`
	Name      string     `gorm:"not null"`
	SprintID  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time
}

func (testPlan) TableName() string { return "test_plans" }

type testSuite struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PlanID    uuid.UUID `gorm:"type:uuid;not null"`
	Name      string    `gorm:"not null"`
	Position  float64   `gorm:"not null"`
	CreatedAt time.Time
}

func (testSuite) TableName() string { return "test_suites" }

type testCase struct {
	ID        uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SuiteID   uuid.UUID  `gorm:"type:uuid;not null"`
	Title     string     `gorm:"not null"`
	Steps     string     `gorm:"type:jsonb;not null"`
	TaskID    *uuid.UUID `gorm:"type:uuid"`
	Position  float64    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (testCase) TableName() string { return "test_cases" }

type testRun struct {
	ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	PlanID      uuid.UUID  `gorm:"type:uuid;not null"`
	Name        string     `gorm:"not null"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (testRun) TableName() string { return "test_runs" }

type testResult struct {
	ID           uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	RunID        uuid.UUID  `gorm:"type:uuid;not null"`
	CaseID       *uuid.UUID `gorm:"type:uuid"`
	Title        string     `gorm:"not null"`
	Steps        string     `gorm:"type:jsonb;not null"`
	Outcome      string     `gorm:"not null"`
	StepOutcomes string     `gorm:"type:jsonb;not null"`
	Comment      *string
	BugTaskID    *uuid.UUID `gorm:"type:uuid"`
	TestedBy     *uuid.UUID `gorm:"type:uuid"`
	UpdatedAt    time.Time
}

func (testResult) TableName() string { return "test_results" }

func steps(s string) []api.TestStep {
	out := []api.TestStep{}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func (c testCase) toAPI() api.TestCase {
	return api.TestCase{Id: c.ID, SuiteId: c.SuiteID, Title: c.Title, Steps: steps(c.Steps), TaskId: c.TaskID, Position: float32(c.Position)}
}

func (r testResult) toAPI() api.TestResult {
	so := []api.TestOutcome{}
	_ = json.Unmarshal([]byte(r.StepOutcomes), &so)
	return api.TestResult{Id: r.ID, RunId: r.RunID, CaseId: r.CaseID, Title: r.Title, Steps: steps(r.Steps), Outcome: api.TestOutcome(r.Outcome),
		StepOutcomes: so, Comment: r.Comment, BugTaskId: r.BugTaskID, TestedBy: r.TestedBy, UpdatedAt: r.UpdatedAt}
}

type outcomeCounts struct{ passed, failed, blocked, notRun int }

func countOutcomes(rs []testResult) outcomeCounts {
	var c outcomeCounts
	for _, r := range rs {
		switch r.Outcome {
		case "passed":
			c.passed++
		case "failed":
			c.failed++
		case "blocked":
			c.blocked++
		default:
			c.notRun++
		}
	}
	return c
}

func (h *Handler) runAPI(r testRun) api.TestRun {
	var rs []testResult
	h.db.Select("outcome").Where("run_id = ?", r.ID).Find(&rs)
	c := countOutcomes(rs)
	return api.TestRun{Id: r.ID, PlanId: r.PlanID, Name: r.Name, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt,
		Passed: c.passed, Failed: c.failed, Blocked: c.blocked, NotRun: c.notRun}
}

func (h *Handler) planAPI(p testPlan) api.TestPlan {
	out := api.TestPlan{Id: p.ID, ProjectId: p.ProjectID, Name: p.Name, SprintId: p.SprintID, CreatedAt: p.CreatedAt}
	var n int64
	h.db.Model(&testCase{}).Where("suite_id IN (?)", h.db.Model(&testSuite{}).Select("id").Where("plan_id = ?", p.ID)).Count(&n)
	out.CaseCount = int(n)
	var last testRun
	if h.db.Where("plan_id = ?", p.ID).Order("created_at DESC").First(&last).Error == nil {
		r := h.runAPI(last)
		out.Passed, out.Failed, out.Blocked, out.NotRun = r.Passed, r.Failed, r.Blocked, r.NotRun
	}
	return out
}

func badReq(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, api.Error{Message: msg})
}

func validName(s string, max int) bool {
	s = strings.TrimSpace(s)
	return s != "" && utf8.RuneCountInString(s) <= max
}

// ListTestPlans implements GET /api/v1/test-plans.
func (h *Handler) ListTestPlans(c *gin.Context) {
	var rows []testPlan
	h.db.Where("project_id = ?", c.Query("projectId")).Order("created_at DESC").Find(&rows)
	out := api.TestPlanList{Data: make([]api.TestPlan, len(rows))}
	for i, p := range rows {
		out.Data[i] = h.planAPI(p)
	}
	c.JSON(http.StatusOK, out)
}

func (h *Handler) readPlan(c *gin.Context, p *testPlan) bool {
	var in api.TestPlanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badReq(c, "invalid request body")
		return false
	}
	var n int64
	h.db.Model(&models.Project{}).Where("id = ?", in.ProjectId).Count(&n)
	if n == 0 || (p.ID != uuid.Nil && in.ProjectId != p.ProjectID) {
		badReq(c, "unknown project")
		return false
	}
	if !h.canWriteProject(c, in.ProjectId) {
		return false
	}
	if !validName(in.Name, 200) {
		badReq(c, "a plan needs a name of 1–200 characters")
		return false
	}
	if in.SprintId != nil {
		var s models.Sprint
		if h.db.First(&s, "id = ? AND project_id = ?", *in.SprintId, in.ProjectId).Error != nil {
			badReq(c, "that sprint isn't in this project")
			return false
		}
	}
	p.ProjectID, p.Name, p.SprintID = in.ProjectId, strings.TrimSpace(in.Name), in.SprintId
	return true
}

// CreateTestPlan implements POST /api/v1/test-plans.
func (h *Handler) CreateTestPlan(c *gin.Context) {
	var p testPlan
	if !h.readPlan(c, &p) {
		return
	}
	h.db.Create(&p)
	c.JSON(http.StatusCreated, h.planAPI(p))
}

func (h *Handler) loadTestPlan(c *gin.Context, id any) (testPlan, bool) {
	var p testPlan
	if err := h.db.First(&p, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "test plan not found"})
		return p, false
	}
	return p, true
}

// GetTestPlan implements GET /api/v1/test-plans/{id}.
func (h *Handler) GetTestPlan(c *gin.Context) {
	p, ok := h.loadTestPlan(c, c.Param("id"))
	if !ok {
		return
	}
	base := h.planAPI(p)
	out := api.TestPlanDetail{Id: base.Id, ProjectId: base.ProjectId, Name: base.Name, SprintId: base.SprintId, CreatedAt: base.CreatedAt,
		CaseCount: base.CaseCount, Passed: base.Passed, Failed: base.Failed, Blocked: base.Blocked, NotRun: base.NotRun,
		Suites: []api.TestSuite{}, Runs: []api.TestRun{}}
	var suites []testSuite
	h.db.Where("plan_id = ?", p.ID).Order("position, created_at").Find(&suites)
	for _, s := range suites {
		var cases []testCase
		h.db.Where("suite_id = ?", s.ID).Order("position, created_at").Find(&cases)
		as := api.TestSuite{Id: s.ID, PlanId: s.PlanID, Name: s.Name, Position: float32(s.Position), Cases: make([]api.TestCase, len(cases))}
		for i, tc := range cases {
			as.Cases[i] = tc.toAPI()
		}
		out.Suites = append(out.Suites, as)
	}
	var runs []testRun
	h.db.Where("plan_id = ?", p.ID).Order("created_at DESC").Limit(50).Find(&runs)
	for _, r := range runs {
		out.Runs = append(out.Runs, h.runAPI(r))
	}
	c.JSON(http.StatusOK, out)
}

// UpdateTestPlan implements PATCH /api/v1/test-plans/{id}.
func (h *Handler) UpdateTestPlan(c *gin.Context) {
	p, ok := h.loadTestPlan(c, c.Param("id"))
	if !ok || !h.readPlan(c, &p) {
		return
	}
	h.db.Save(&p)
	c.JSON(http.StatusOK, h.planAPI(p))
}

// DeleteTestPlan implements DELETE /api/v1/test-plans/{id}.
func (h *Handler) DeleteTestPlan(c *gin.Context) {
	p, ok := h.loadTestPlan(c, c.Param("id"))
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	suiteIDs := h.db.Model(&testSuite{}).Select("id").Where("plan_id = ?", p.ID)
	runIDs := h.db.Model(&testRun{}).Select("id").Where("plan_id = ?", p.ID)
	h.db.Where("run_id IN (?)", runIDs).Delete(&testResult{})
	h.db.Where("plan_id = ?", p.ID).Delete(&testRun{})
	h.db.Where("suite_id IN (?)", suiteIDs).Delete(&testCase{})
	h.db.Where("plan_id = ?", p.ID).Delete(&testSuite{})
	h.db.Delete(&p)
	c.Status(http.StatusNoContent)
}

// --- suites ---

func (h *Handler) readSuite(c *gin.Context, s *testSuite) bool {
	var in api.TestSuiteInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badReq(c, "invalid request body")
		return false
	}
	p, ok := h.loadTestPlan(c, in.PlanId)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return false
	}
	if !validName(in.Name, 200) {
		badReq(c, "a suite needs a name of 1–200 characters")
		return false
	}
	s.PlanID, s.Name = p.ID, strings.TrimSpace(in.Name)
	if in.Position != nil {
		s.Position = float64(*in.Position)
	} else if s.ID == uuid.Nil {
		var max float64
		h.db.Model(&testSuite{}).Where("plan_id = ?", p.ID).Select("COALESCE(MAX(position), 0)").Scan(&max)
		s.Position = max + 1000
	}
	return true
}

func suiteAPI(s testSuite) api.TestSuite {
	return api.TestSuite{Id: s.ID, PlanId: s.PlanID, Name: s.Name, Position: float32(s.Position), Cases: []api.TestCase{}}
}

// CreateTestSuite implements POST /api/v1/test-suites.
func (h *Handler) CreateTestSuite(c *gin.Context) {
	var s testSuite
	if !h.readSuite(c, &s) {
		return
	}
	h.db.Create(&s)
	c.JSON(http.StatusCreated, suiteAPI(s))
}

// UpdateTestSuite implements PATCH /api/v1/test-suites/{id}.
func (h *Handler) UpdateTestSuite(c *gin.Context) {
	var s testSuite
	if h.db.First(&s, "id = ?", c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "suite not found"})
		return
	}
	if !h.readSuite(c, &s) {
		return
	}
	h.db.Save(&s)
	c.JSON(http.StatusOK, suiteAPI(s))
}

// DeleteTestSuite implements DELETE /api/v1/test-suites/{id}.
func (h *Handler) DeleteTestSuite(c *gin.Context) {
	var s testSuite
	if h.db.First(&s, "id = ?", c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "suite not found"})
		return
	}
	p, ok := h.loadTestPlan(c, s.PlanID)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	h.db.Where("suite_id = ?", s.ID).Delete(&testCase{})
	h.db.Delete(&s)
	c.Status(http.StatusNoContent)
}

// --- cases ---

func (h *Handler) readCase(c *gin.Context, tc *testCase) bool {
	var in api.TestCaseInput
	if err := c.ShouldBindJSON(&in); err != nil {
		badReq(c, "invalid request body")
		return false
	}
	var s testSuite
	if h.db.First(&s, "id = ?", in.SuiteId).Error != nil {
		badReq(c, "unknown suite")
		return false
	}
	p, ok := h.loadTestPlan(c, s.PlanID)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return false
	}
	if !validName(in.Title, 300) {
		badReq(c, "a test case needs a title of 1–300 characters")
		return false
	}
	if in.Steps != nil {
		if len(*in.Steps) > 100 {
			badReq(c, "up to 100 steps")
			return false
		}
		for _, st := range *in.Steps {
			if utf8.RuneCountInString(st.Action) > 2000 || utf8.RuneCountInString(st.Expected) > 2000 {
				badReq(c, "steps are up to 2000 characters")
				return false
			}
		}
		b, _ := json.Marshal(*in.Steps)
		tc.Steps = string(b)
	}
	if tc.Steps == "" {
		tc.Steps = "[]"
	}
	tc.TaskID = nil
	if in.TaskId != nil {
		var t models.Task
		if h.db.First(&t, "id = ? AND project_id = ?", *in.TaskId, p.ProjectID).Error != nil {
			badReq(c, "that item isn't in this project")
			return false
		}
		tc.TaskID = &t.ID
	}
	tc.SuiteID, tc.Title = s.ID, strings.TrimSpace(in.Title)
	if in.Position != nil {
		tc.Position = float64(*in.Position)
	} else if tc.ID == uuid.Nil {
		var max float64
		h.db.Model(&testCase{}).Where("suite_id = ?", s.ID).Select("COALESCE(MAX(position), 0)").Scan(&max)
		tc.Position = max + 1000
	}
	return true
}

// CreateTestCase implements POST /api/v1/test-cases.
func (h *Handler) CreateTestCase(c *gin.Context) {
	var tc testCase
	if !h.readCase(c, &tc) {
		return
	}
	h.db.Create(&tc)
	c.JSON(http.StatusCreated, tc.toAPI())
}

// UpdateTestCase implements PATCH /api/v1/test-cases/{id}.
func (h *Handler) UpdateTestCase(c *gin.Context) {
	var tc testCase
	if h.db.First(&tc, "id = ?", c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "test case not found"})
		return
	}
	if !h.readCase(c, &tc) {
		return
	}
	h.db.Save(&tc)
	c.JSON(http.StatusOK, tc.toAPI())
}

// DeleteTestCase implements DELETE /api/v1/test-cases/{id}.
func (h *Handler) DeleteTestCase(c *gin.Context) {
	var tc testCase
	if h.db.First(&tc, "id = ?", c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "test case not found"})
		return
	}
	var s testSuite
	h.db.First(&s, "id = ?", tc.SuiteID)
	p, ok := h.loadTestPlan(c, s.PlanID)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	h.db.Delete(&tc)
	c.Status(http.StatusNoContent)
}

// --- runs and results ---

func (h *Handler) runDetail(r testRun) api.TestRunDetail {
	b := h.runAPI(r)
	out := api.TestRunDetail{Id: b.Id, PlanId: b.PlanId, Name: b.Name, CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt, CompletedAt: b.CompletedAt,
		Passed: b.Passed, Failed: b.Failed, Blocked: b.Blocked, NotRun: b.NotRun, Results: []api.TestResult{}}
	var rs []testResult
	h.db.Where("run_id = ?", r.ID).Order("title").Find(&rs)
	for _, x := range rs {
		out.Results = append(out.Results, x.toAPI())
	}
	return out
}

// CreateTestRun implements POST /api/v1/test-runs.
func (h *Handler) CreateTestRun(c *gin.Context) {
	var in struct {
		PlanID  uuid.UUID  `json:"planId"`
		SuiteID *uuid.UUID `json:"suiteId"`
		Name    *string    `json:"name"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		badReq(c, "invalid request body")
		return
	}
	p, ok := h.loadTestPlan(c, in.PlanID)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	q := h.db.Where("suite_id IN (?)", h.db.Model(&testSuite{}).Select("id").Where("plan_id = ?", p.ID))
	if in.SuiteID != nil {
		q = h.db.Where("suite_id = ?", *in.SuiteID)
	}
	var cases []testCase
	q.Order("position").Find(&cases)
	if len(cases) == 0 {
		badReq(c, "there are no test cases to run")
		return
	}
	me, _ := h.actor(c)
	name := fmt.Sprintf("%s — %s", p.Name, time.Now().Format("2 Jan 15:04"))
	if in.Name != nil && validName(*in.Name, 100) {
		name = strings.TrimSpace(*in.Name)
	}
	r := testRun{PlanID: p.ID, Name: name}
	if me != uuid.Nil {
		r.CreatedBy = &me
	}
	h.db.Create(&r)
	for _, tc := range cases {
		id := tc.ID
		so, _ := json.Marshal(make([]string, len(steps(tc.Steps))))
		so = []byte(strings.ReplaceAll(string(so), `""`, `"not_run"`))
		h.db.Create(&testResult{RunID: r.ID, CaseID: &id, Title: tc.Title, Steps: tc.Steps, Outcome: "not_run", StepOutcomes: string(so)})
	}
	c.JSON(http.StatusCreated, h.runDetail(r))
}

func (h *Handler) loadRun(c *gin.Context, id any) (testRun, testPlan, bool) {
	var r testRun
	if h.db.First(&r, "id = ?", id).Error != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "test run not found"})
		return r, testPlan{}, false
	}
	p, ok := h.loadTestPlan(c, r.PlanID)
	return r, p, ok
}

// GetTestRun implements GET /api/v1/test-runs/{id}.
func (h *Handler) GetTestRun(c *gin.Context) {
	if r, _, ok := h.loadRun(c, c.Param("id")); ok {
		c.JSON(http.StatusOK, h.runDetail(r))
	}
}

// DeleteTestRun implements DELETE /api/v1/test-runs/{id}.
func (h *Handler) DeleteTestRun(c *gin.Context) {
	r, p, ok := h.loadRun(c, c.Param("id"))
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	h.db.Where("run_id = ?", r.ID).Delete(&testResult{})
	h.db.Delete(&r)
	c.Status(http.StatusNoContent)
}

func overall(so []api.TestOutcome) api.TestOutcome {
	if len(so) == 0 {
		return api.TestOutcomeNotRun
	}
	all := true
	for _, o := range so {
		if o == api.TestOutcomeFailed {
			return api.TestOutcomeFailed
		}
	}
	for _, o := range so {
		if o == api.TestOutcomeBlocked {
			return api.TestOutcomeBlocked
		}
		all = all && o == api.TestOutcomePassed
	}
	if all {
		return api.TestOutcomePassed
	}
	return api.TestOutcomeNotRun
}

func (h *Handler) loadResult(c *gin.Context) (testResult, testRun, testPlan, bool) {
	var x testResult
	if h.db.First(&x, "id = ?", c.Param("id")).Error != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "result not found"})
		return x, testRun{}, testPlan{}, false
	}
	r, p, ok := h.loadRun(c, x.RunID)
	return x, r, p, ok
}

// UpdateTestResult implements PATCH /api/v1/test-results/{id}.
func (h *Handler) UpdateTestResult(c *gin.Context) {
	x, r, p, ok := h.loadResult(c)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	var in api.TestResultUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		badReq(c, "invalid request body")
		return
	}
	if in.Outcome != nil && !in.Outcome.Valid() {
		badReq(c, "outcome is not_run, passed, failed or blocked")
		return
	}
	if in.StepOutcomes != nil {
		if len(*in.StepOutcomes) != len(steps(x.Steps)) {
			badReq(c, "give one outcome per step")
			return
		}
		for _, o := range *in.StepOutcomes {
			if !o.Valid() {
				badReq(c, "step outcomes are not_run, passed, failed or blocked")
				return
			}
		}
		b, _ := json.Marshal(*in.StepOutcomes)
		x.StepOutcomes = string(b)
		if in.Outcome == nil {
			x.Outcome = string(overall(*in.StepOutcomes))
		}
	}
	if in.Outcome != nil {
		x.Outcome = string(*in.Outcome)
	}
	if in.Comment != nil {
		if utf8.RuneCountInString(*in.Comment) > 4000 {
			badReq(c, "comments are up to 4000 characters")
			return
		}
		cm := strings.TrimSpace(*in.Comment)
		x.Comment = &cm
		if cm == "" {
			x.Comment = nil
		}
	}
	if me, _ := h.actor(c); me != uuid.Nil {
		x.TestedBy = &me
	}
	h.db.Save(&x)
	// The run is complete once nothing is left not run.
	var left int64
	h.db.Model(&testResult{}).Where("run_id = ? AND outcome = ?", r.ID, "not_run").Count(&left)
	if left == 0 && r.CompletedAt == nil {
		now := time.Now()
		h.db.Model(&testRun{}).Where("id = ?", r.ID).Update("completed_at", now)
	} else if left > 0 && r.CompletedAt != nil {
		h.db.Model(&testRun{}).Where("id = ?", r.ID).Update("completed_at", nil)
	}
	c.JSON(http.StatusOK, x.toAPI())
}

// CreateBugFromResult implements POST /api/v1/test-results/{id}/bug.
func (h *Handler) CreateBugFromResult(c *gin.Context) {
	x, r, p, ok := h.loadResult(c)
	if !ok || !h.canWriteProject(c, p.ProjectID) {
		return
	}
	so := []api.TestOutcome{}
	_ = json.Unmarshal([]byte(x.StepOutcomes), &so)
	var b strings.Builder
	fmt.Fprintf(&b, "Found by test run **%s**, case “%s”.\n\n## Steps to reproduce\n", r.Name, x.Title)
	for i, st := range steps(x.Steps) {
		mark := ""
		if i < len(so) && so[i] != api.TestOutcomeNotRun {
			mark = fmt.Sprintf(" — **%s**", so[i])
		}
		fmt.Fprintf(&b, "%d. %s%s\n   - Expected: %s\n", i+1, st.Action, mark, st.Expected)
	}
	if x.Comment != nil {
		fmt.Fprintf(&b, "\n## Notes\n%s\n", *x.Comment)
	}
	desc := b.String()
	t := models.Task{ProjectID: p.ProjectID, Type: api.TaskTypeBug, Title: excerpt("Failed: "+x.Title, 300), Description: &desc,
		Priority: api.TaskPriorityHigh, Status: api.TaskStatusTodo}
	if err := h.resolveState(&t, nil, true); err != nil {
		badReq(c, err.Error())
		return
	}
	var maxNumber int
	var maxRank float64
	h.db.Model(&models.Task{}).Where("project_id = ?", p.ProjectID).Select("COALESCE(MAX(number), 0)").Scan(&maxNumber)
	h.db.Model(&models.Task{}).Where("project_id = ?", p.ProjectID).Select("COALESCE(MAX(rank), 0)").Scan(&maxRank)
	t.Number, t.Rank = maxNumber+1, maxRank+rankStep
	if err := h.db.Create(&t).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to file the bug", Details: strptr(err.Error())})
		return
	}
	if x.CaseID != nil {
		var tc testCase
		if h.db.First(&tc, "id = ?", *x.CaseID).Error == nil && tc.TaskID != nil {
			_ = h.replaceRelated(&t, []uuid.UUID{*tc.TaskID})
		}
	}
	h.db.Model(&testResult{}).Where("id = ?", x.ID).Update("bug_task_id", t.ID)
	h.logTask(c, "created", t.ID, nil)
	tasks := []models.Task{t}
	if err := h.enrichTasks(tasks); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load the bug"})
		return
	}
	c.JSON(http.StatusCreated, tasks[0].ToAPI())
}
