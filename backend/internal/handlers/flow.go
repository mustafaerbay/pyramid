package handlers

import (
	"errors"
	"math"
	"net/http"
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

func round1(v float64) float64 { return math.Round(v*10) / 10 }

// GetProjectFlow implements GET /api/v1/projects/{id}/flow.
func (h *Handler) GetProjectFlow(c *gin.Context) {
	var p models.Project
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "project not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load project", Details: strptr(err.Error())})
		}
		return
	}
	days := 30
	if v, err := strconv.Atoi(c.Query("days")); err == nil {
		days = clamp(v, 7, 180)
	}
	fail := func(err error) {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to build flow", Details: strptr(err.Error())})
	}
	var tasks []models.Task
	if err := h.db.Where("project_id = ?", p.ID).Find(&tasks).Error; err != nil {
		fail(err)
		return
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	timelines, err := h.statusTimelines(ids)
	if err != nil {
		fail(err)
		return
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	first := today.AddDate(0, 0, -(days - 1))
	out := api.ProjectFlow{Days: make([]api.FlowDay, 0, days), Completed: []api.FlowItem{}}
	for d := first; !d.After(today); d = d.AddDate(0, 0, 1) {
		end := d.AddDate(0, 0, 1)
		counts := map[string]int{}
		for _, t := range tasks {
			if !t.CreatedAt.Before(end) {
				continue
			}
			st := statusAt(string(t.Status), timelines[t.ID], end)
			if st != string(api.TaskStatusCancelled) {
				counts[st]++
			}
		}
		out.Days = append(out.Days, api.FlowDay{Date: openapi_types.Date{Time: d}, Counts: counts})
	}

	for _, t := range tasks {
		if t.Status != api.TaskStatusDone || t.CompletedAt == nil || t.CompletedAt.Before(first) {
			continue
		}
		item := api.FlowItem{
			TaskId: t.ID, Number: t.Number, Title: t.Title, Type: t.Type, CompletedAt: *t.CompletedAt,
			LeadDays: round1(t.CompletedAt.Sub(t.CreatedAt).Hours() / 24),
		}
		ch := timelines[t.ID]
		for i := 0; i+1 < len(ch); i += 2 {
			if ch[i+1].status == string(api.TaskStatusInProgress) {
				v := round1(t.CompletedAt.Sub(ch[i+1].at).Hours() / 24)
				item.CycleDays = &v
				break
			}
		}
		out.Completed = append(out.Completed, item)
	}
	sort.Slice(out.Completed, func(i, j int) bool { return out.Completed[i].CompletedAt.After(out.Completed[j].CompletedAt) })
	c.JSON(http.StatusOK, out)
}
