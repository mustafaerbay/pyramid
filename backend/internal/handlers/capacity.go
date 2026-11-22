package handlers

import (
	"net/http"
	"regexp"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// monthPlan is the man-days planned in one calendar month, per person and
// per project, after spreading every task assignment over its dates (see
// models.SpreadManDays). Cancelled tasks and cancelled projects don't count;
// done tasks do (the work still happened in that month).
type monthPlan struct {
	byPerson  map[uuid.UUID]float64
	byProject map[uuid.UUID]float64
	total     float64
}

func (h *Handler) monthPlan(month time.Time) (monthPlan, error) {
	plan := monthPlan{byPerson: map[uuid.UUID]float64{}, byProject: map[uuid.UUID]float64{}}
	var rows []struct {
		PersonID  uuid.UUID
		ProjectID uuid.UUID
		ManDays   float64
		StartDate *time.Time
		DueDate   *time.Time
	}
	err := h.db.Table("task_assignees AS ta").
		Select("ta.person_id, t.project_id, ta.man_days, t.start_date, t.due_date").
		Joins("JOIN tasks t ON t.id = ta.task_id").
		Joins("JOIN projects p ON p.id = t.project_id").
		Where("ta.man_days > 0").
		Where("t.status <> ?", "cancelled").
		Where("p.status <> ?", "cancelled").
		Where("t.start_date IS NOT NULL OR t.due_date IS NOT NULL").
		Scan(&rows).Error
	if err != nil {
		return plan, err
	}
	for _, r := range rows {
		md := models.SpreadManDays(r.ManDays, r.StartDate, r.DueDate, month)
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
	if err := h.db.Where("status = ?", api.PersonStatusActive).Order("first_name, last_name").Find(&people).Error; err != nil {
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
		cap := p.Capacity()
		out.CapacityManDays += cap
		out.People = append(out.People, api.PersonCapacity{
			PersonId: p.ID, CapacityManDays: cap, PlannedManDays: plan.byPerson[p.ID],
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
