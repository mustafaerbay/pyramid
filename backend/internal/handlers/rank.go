package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Manual order: tasks.rank, lower first. A move takes the midpoint between
// the neighbours; when gaps get too small the project is spread out again.

const (
	rankStep   = 1000.0
	minRankGap = 1e-6
)

func (h *Handler) respreadRanks(projectID any) error {
	var ids []string
	if err := h.db.Model(&models.Task{}).Where("project_id = ?", projectID).
		Order("rank ASC, number ASC").Pluck("id", &ids).Error; err != nil {
		return err
	}
	for i, id := range ids {
		if err := h.db.Model(&models.Task{}).Where("id = ?", id).Update("rank", float64(i+1)*rankStep).Error; err != nil {
			return err
		}
	}
	return nil
}

// newRank is where a task goes right after `after` (nil = first).
func (h *Handler) newRank(t models.Task, after *models.Task) (float64, bool, error) {
	q := h.db.Model(&models.Task{}).Where("project_id = ? AND id <> ?", t.ProjectID, t.ID)
	if after == nil {
		var min float64
		var n int64
		q.Count(&n)
		if n == 0 {
			return rankStep, true, nil
		}
		if err := q.Select("MIN(rank)").Scan(&min).Error; err != nil {
			return 0, false, err
		}
		return min - rankStep, true, nil
	}
	var next []float64
	if err := q.Where("rank > ?", after.Rank).Order("rank ASC").Limit(1).Pluck("rank", &next).Error; err != nil {
		return 0, false, err
	}
	if len(next) == 0 {
		return after.Rank + rankStep, true, nil
	}
	if next[0]-after.Rank < minRankGap {
		return 0, false, nil
	}
	return (after.Rank + next[0]) / 2, true, nil
}

// MoveTask implements POST /api/v1/tasks/{id}/move.
func (h *Handler) MoveTask(c *gin.Context) {
	if !h.canWriteTask(c) {
		return
	}
	t, ok := h.loadTask(c)
	if !ok {
		return
	}
	var in api.TaskMove
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	var after *models.Task
	if in.AfterId != nil {
		if *in.AfterId == t.ID {
			c.JSON(http.StatusBadRequest, api.Error{Message: "a task can't go after itself"})
			return
		}
		var a models.Task
		if err := h.db.First(&a, "id = ? AND project_id = ?", *in.AfterId, t.ProjectID).Error; err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "afterId must be a task in the same project"})
			return
		}
		after = &a
	}
	rank, ok, err := h.newRank(t, after)
	if err == nil && !ok {
		// No room between the neighbours: spread everything out and retry.
		if err = h.respreadRanks(t.ProjectID); err == nil {
			if after != nil {
				err = h.db.First(after, "id = ?", after.ID).Error
			}
			if err == nil {
				rank, _, err = h.newRank(t, after)
			}
		}
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to move task", Details: strptr(err.Error())})
		return
	}
	if err := h.db.Model(&t).Update("rank", rank).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to move task", Details: strptr(err.Error())})
		return
	}
	c.Request.Method = http.MethodGet
	h.GetTask(c)
}
