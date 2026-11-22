package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// ListLeaveRequests implements GET /api/v1/leave-requests.
func (h *Handler) ListLeaveRequests(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.LeaveRequest{})
	// Repeatable ?status=pending&status=approved — a leave request matches
	// if its status is any of the given values. QueryArray returns an empty
	// slice (not an error) when the key is absent, so this is a no-op filter
	// by default, same as the old single-value check. Mirrors the type
	// filter in contributions.go.
	if statuses := c.QueryArray("status"); len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	if p := c.Query("personId"); p != "" {
		q = q.Where("person_id = ?", p)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count leave requests", Details: strptr(err.Error())})
		return
	}

	var requests []models.LeaveRequest
	if err := q.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&requests).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list leave requests", Details: strptr(err.Error())})
		return
	}

	data := make([]api.LeaveRequest, len(requests))
	for i := range requests {
		data[i] = requests[i].ToAPI()
	}

	c.JSON(http.StatusOK, api.LeaveRequestList{
		Data: data,
		Pagination: api.Pagination{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	})
}

// CreateLeaveRequest implements POST /api/v1/leave-requests.
func (h *Handler) CreateLeaveRequest(c *gin.Context) {
	var in api.LeaveRequestCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	l := models.NewLeaveRequestFromCreate(in)
	if err := h.db.Create(&l).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create leave request", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, l.ToAPI())
}

// GetLeaveRequest implements GET /api/v1/leave-requests/{id}.
func (h *Handler) GetLeaveRequest(c *gin.Context) {
	var l models.LeaveRequest
	if err := h.db.First(&l, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "leave request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave request", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, l.ToAPI())
}

// UpdateLeaveRequest implements PATCH /api/v1/leave-requests/{id}.
func (h *Handler) UpdateLeaveRequest(c *gin.Context) {
	var l models.LeaveRequest
	if err := h.db.First(&l, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "leave request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load leave request", Details: strptr(err.Error())})
		return
	}

	var in api.LeaveRequestUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	l.ApplyUpdate(in)
	if err := h.db.Save(&l).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update leave request", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, l.ToAPI())
}

// DeleteLeaveRequest implements DELETE /api/v1/leave-requests/{id}.
func (h *Handler) DeleteLeaveRequest(c *gin.Context) {
	res := h.db.Delete(&models.LeaveRequest{}, "id = ?", c.Param("id"))
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete leave request", Details: strptr(res.Error.Error())})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "leave request not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
