package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"
)

// monthPlan is the man-days planned in one calendar month, per person and
// per project, after spreading every task assignment over its dates (see
// models.SpreadManDays). Cancelled tasks and cancelled projects don't count;
// done tasks do (the work still happened in that month).
type monthPlan struct {
	month     time.Time
	byPerson  map[uuid.UUID]float64
	byProject map[uuid.UUID]float64
	total     float64
	leave     map[uuid.UUID][]models.DateRange // everyone's approved leave
}

// capacity is p's capacity in the plan's month after approved leave, and
// the number of leave days taken out.
func (mp monthPlan) capacity(p models.Person) (float64, int) {
	leave := models.LeaveDaysInMonth(mp.leave[p.ID], mp.month)
	return models.MonthCapacity(p.Capacity(), models.WorkingDaysInMonth(mp.month), leave), leave
}

// approvedLeave returns the approved leave of the given people (everyone
// when personIDs is nil), keyed by person. Pending, rejected and cancelled
// requests don't count. A task can span months, so this isn't limited to
// one month; leave_requests stays small.
func (h *Handler) approvedLeave(personIDs []uuid.UUID) (map[uuid.UUID][]models.DateRange, error) {
	q := h.db.Where("status = ?", api.LeaveStatusApproved)
	if personIDs != nil {
		if len(personIDs) == 0 {
			return map[uuid.UUID][]models.DateRange{}, nil
		}
		q = q.Where("person_id IN ?", personIDs)
	}
	var rows []models.LeaveRequest
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[uuid.UUID][]models.DateRange{}
	for _, l := range rows {
		out[l.PersonID] = append(out[l.PersonID], models.DateRange{From: l.StartDate, To: l.EndDate})
	}
	return out, nil
}

func (h *Handler) monthPlan(month time.Time) (monthPlan, error) {
	month = models.MonthStart(month)
	plan := monthPlan{month: month, byPerson: map[uuid.UUID]float64{}, byProject: map[uuid.UUID]float64{}}
	leave, err := h.approvedLeave(nil)
	if err != nil {
		return plan, err
	}
	plan.leave = leave
	var rows []struct {
		PersonID  uuid.UUID
		ProjectID uuid.UUID
		ManDays   float64
		StartDate *time.Time
		DueDate   *time.Time
	}
	err = h.db.Table("task_assignees AS ta").
		Select("ta.person_id, t.project_id, ta.man_days, t.start_date, t.due_date").
		Joins("JOIN tasks t ON t.id = ta.task_id").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("ta.man_days > 0").
		Where("t.status <> ?", "cancelled").
		Where("p.status <> ?", "cancelled").
		Where("t.start_date IS NOT NULL OR t.due_date IS NOT NULL").
		Where(countedWork).
		Scan(&rows).Error
	if err != nil {
		return plan, err
	}
	for _, r := range rows {
		md := models.SpreadManDaysAroundLeave(r.ManDays, r.StartDate, r.DueDate, plan.leave[r.PersonID], month)
		if md == 0 {
			continue
		}
		plan.byPerson[r.PersonID] += md
		plan.byProject[r.ProjectID] += md
		plan.total += md
	}
	return plan, nil
}

var monthParam = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

// GetCapacity implements GET /api/v1/capacity?month=YYYY-MM.
func (h *Handler) GetCapacity(c *gin.Context) {
	month := models.MonthStart(time.Now().UTC())
	if m := c.Query("month"); m != "" {
		if !monthParam.MatchString(m) {
			c.JSON(http.StatusBadRequest, api.Error{Message: "month must be YYYY-MM"})
			return
		}
		parsed, err := time.Parse("2006-01", m)
		if err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "month must be YYYY-MM"})
			return
		}
		month = parsed
	}

	plan, err := h.monthPlan(month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to compute planned man-days", Details: strptr(err.Error())})
		return
	}

	var people []models.Person
	// People on leave stay in: their approved leave takes their capacity out.
	if err := h.db.Where("status IN ?", []api.PersonStatus{api.PersonStatusActive, api.PersonStatusOnLeave}).Order("first_name, last_name").Find(&people).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load people", Details: strptr(err.Error())})
		return
	}

	out := api.CapacityReport{
		Month:          month.Format("2006-01"),
		WorkingDays:    models.WorkingDaysInMonth(month),
		PlannedManDays: plan.total,
		People:         make([]api.PersonCapacity, 0, len(people)),
		Projects:       make([]api.ProjectLoad, 0, len(plan.byProject)),
	}
	for _, p := range people {
		cap, leave := plan.capacity(p)
		out.CapacityManDays += cap
		out.People = append(out.People, api.PersonCapacity{
			PersonId: p.ID, CapacityManDays: cap, LeaveDays: leave, PlannedManDays: plan.byPerson[p.ID],
		})
	}
	for id, md := range plan.byProject {
		out.Projects = append(out.Projects, api.ProjectLoad{ProjectId: id, PlannedManDays: md})
	}
	sort.Slice(out.Projects, func(i, j int) bool {
		return out.Projects[i].PlannedManDays > out.Projects[j].PlannedManDays
	})
	c.JSON(http.StatusOK, out)
}

// GetPersonWorkload implements GET /api/v1/people/{id}/workload?year=YYYY:
// the person's planned man-days for each month of the year, by project and
// by task, with the same spreading rule as GetCapacity.
func (h *Handler) GetPersonWorkload(c *gin.Context) {
	var person models.Person
	if err := h.db.First(&person, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "person not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load person", Details: strptr(err.Error())})
		return
	}

	year := time.Now().UTC().Year()
	if y := c.Query("year"); y != "" {
		n, err := strconv.Atoi(y)
		if err != nil || n < 2000 || n > 2100 {
			c.JSON(http.StatusBadRequest, api.Error{Message: "year must be between 2000 and 2100"})
			return
		}
		year = n
	}

	var rows []struct {
		TaskID    uuid.UUID
		ProjectID uuid.UUID
		Title     string
		Status    api.TaskStatus
		ManDays   float64
		StartDate *time.Time
		DueDate   *time.Time
	}
	err := h.db.Table("task_assignees AS ta").
		Select("t.id AS task_id, t.project_id, t.title, t.status, ta.man_days, t.start_date, t.due_date").
		Joins("JOIN tasks t ON t.id = ta.task_id").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("ta.person_id = ?", person.ID).
		Where("ta.man_days > 0").
		Where("t.status <> ?", "cancelled").
		Where("p.status <> ?", "cancelled").
		Where("t.start_date IS NOT NULL OR t.due_date IS NOT NULL").
		Where(countedWork).
		Order("COALESCE(t.start_date, t.due_date)").
		Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load workload", Details: strptr(err.Error())})
		return
	}

	leave, err := h.approvedLeave([]uuid.UUID{person.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave", Details: strptr(err.Error())})
		return
	}

	toDate := func(t *time.Time) *openapi_types.Date {
		if t == nil {
			return nil
		}
		return &openapi_types.Date{Time: *t}
	}

	out := api.PersonWorkload{
		PersonId:        person.ID,
		Year:            year,
		CapacityManDays: person.Capacity(),
		Months:          make([]api.PersonWorkloadMonth, 12),
	}
	for i := range out.Months {
		month := time.Date(year, time.Month(i+1), 1, 0, 0, 0, 0, time.UTC)
		working := models.WorkingDaysInMonth(month)
		leaveDays := models.LeaveDaysInMonth(leave[person.ID], month)
		m := api.PersonWorkloadMonth{
			Month:           month.Format("2006-01"),
			WorkingDays:     working,
			LeaveDays:       leaveDays,
			CapacityManDays: models.MonthCapacity(person.Capacity(), working, leaveDays),
			Projects:        []api.ProjectLoad{},
			Tasks:           []api.PersonWorkloadTask{},
		}
		byProject := map[uuid.UUID]float64{}
		for _, r := range rows {
			md := models.SpreadManDaysAroundLeave(r.ManDays, r.StartDate, r.DueDate, leave[person.ID], month)
			if md == 0 {
				continue
			}
			m.PlannedManDays += md
			if _, seen := byProject[r.ProjectID]; !seen {
				m.Projects = append(m.Projects, api.ProjectLoad{ProjectId: r.ProjectID})
			}
			byProject[r.ProjectID] += md
			m.Tasks = append(m.Tasks, api.PersonWorkloadTask{
				TaskId: r.TaskID, ProjectId: r.ProjectID, Title: r.Title, Status: r.Status,
				ManDays: r.ManDays, MonthManDays: md,
				StartDate: toDate(r.StartDate), DueDate: toDate(r.DueDate),
				AvailableDays: availableDays(r.StartDate, r.DueDate, leave[person.ID]),
			})
		}
		for j := range m.Projects {
			m.Projects[j].PlannedManDays = byProject[m.Projects[j].ProjectId]
		}
		sort.Slice(m.Projects, func(a, b int) bool { return m.Projects[a].PlannedManDays > m.Projects[b].PlannedManDays })
		out.Months[i] = m
	}
	c.JSON(http.StatusOK, out)
}

// availableDays is models.AvailableDays as an API value (nil without a
// start and due date).
func availableDays(start, due *time.Time, leave []models.DateRange) *int {
	n, ok := models.AvailableDays(start, due, leave)
	if !ok {
		return nil
	}
	return &n
}

// attachAvailability fills each task's AvailableDays: per assignee, the
// working days between start and due they aren't on approved leave.
func (h *Handler) attachAvailability(tasks ...*models.Task) error {
	var people []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, t := range tasks {
		for _, id := range t.AssigneeIDs {
			if !seen[id] {
				seen[id] = true
				people = append(people, id)
			}
		}
	}
	if people == nil {
		people = []uuid.UUID{}
	}
	leave, err := h.approvedLeave(people)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		t.AvailableDays = make(map[uuid.UUID]*int, len(t.AssigneeIDs))
		for _, id := range t.AssigneeIDs {
			t.AvailableDays[id] = availableDays(t.StartDate, t.DueDate, leave[id])
		}
	}
	return nil
}

// countedWork keeps tasks whose man-days count toward capacity: a task with
// (non-cancelled) subtasks is just a container, its subtasks carry the work.
const countedWork = "NOT EXISTS (SELECT 1 FROM tasks sub WHERE sub.parent_id = t.id AND sub.status <> 'cancelled')"
