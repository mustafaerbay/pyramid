package handlers

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

const maxCommentLength = 5000

func commentBody(c *gin.Context) (string, bool) {
	var in api.TaskCommentInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return "", false
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a comment can't be empty"})
		return "", false
	}
	if utf8.RuneCountInString(body) > maxCommentLength {
		c.JSON(http.StatusBadRequest, api.Error{Message: "comments can be at most 5000 characters"})
		return "", false
	}
	return body, true
}

func (h *Handler) loadTask(c *gin.Context) (models.Task, bool) {
	var t models.Task
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "task not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load task", Details: strptr(err.Error())})
		}
		return t, false
	}
	return t, true
}

// ListTaskComments implements GET /api/v1/tasks/{id}/comments.
func (h *Handler) ListTaskComments(c *gin.Context) {
	t, ok := h.loadTask(c)
	if !ok {
		return
	}
	var rows []models.TaskComment
	if err := h.db.Where("task_id = ?", t.ID).Order("created_at, id").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load comments", Details: strptr(err.Error())})
		return
	}
	out := make([]api.TaskComment, len(rows))
	for i, r := range rows {
		out[i] = r.ToAPI()
	}
	c.JSON(http.StatusOK, api.TaskCommentList{Data: out})
}

// CreateTaskComment implements POST /api/v1/tasks/{id}/comments.
func (h *Handler) CreateTaskComment(c *gin.Context) {
	t, ok := h.loadTask(c)
	if !ok {
		return
	}
	body, ok := commentBody(c)
	if !ok {
		return
	}
	cm := models.TaskComment{TaskID: t.ID, Body: body}
	if u, ok := currentUser(c); ok {
		pid := u.PersonID
		cm.AuthorID = &pid
	}
	if err := h.db.Create(&cm).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save comment", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, cm.ToAPI())
}

func (h *Handler) loadComment(c *gin.Context) (models.TaskComment, bool) {
	var cm models.TaskComment
	if err := h.db.First(&cm, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "comment not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load comment", Details: strptr(err.Error())})
		}
		return cm, false
	}
	return cm, true
}

func (h *Handler) isAuthor(c *gin.Context, cm models.TaskComment) bool {
	if h.cfg.AuthOff {
		return true
	}
	u, ok := currentUser(c)
	return ok && cm.AuthorID != nil && *cm.AuthorID == u.PersonID
}

// UpdateTaskComment implements PATCH /api/v1/task-comments/{id}.
func (h *Handler) UpdateTaskComment(c *gin.Context) {
	cm, ok := h.loadComment(c)
	if !ok {
		return
	}
	if !h.isAuthor(c, cm) {
		c.JSON(http.StatusForbidden, api.Error{Message: "you can only edit your own comments"})
		return
	}
	body, ok := commentBody(c)
	if !ok {
		return
	}
	cm.Body = body
	if err := h.db.Save(&cm).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save comment", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, cm.ToAPI())
}

// DeleteTaskComment implements DELETE /api/v1/task-comments/{id}.
func (h *Handler) DeleteTaskComment(c *gin.Context) {
	cm, ok := h.loadComment(c)
	if !ok {
		return
	}
	if !h.isAuthor(c, cm) && !h.hasRole(c, api.SystemRoleManager) {
		c.JSON(http.StatusForbidden, api.Error{Message: "only the author, managers and admins can delete a comment"})
		return
	}
	if err := h.db.Delete(&cm).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete comment", Details: strptr(err.Error())})
		return
	}
	c.Status(http.StatusNoContent)
}
