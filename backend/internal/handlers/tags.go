package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Tags are stored in the schema's labels / task_labels tables as global
// labels (project_id NULL), so one tag is shared by every project. Matching
// is case-insensitive: the first spelling saved is the one everyone sees.

const (
	maxTagsPerTask = 20
	maxTagLength   = 40
)

// normalizeTags trims each tag, drops case-insensitive duplicates (first
// spelling wins) and enforces the length and count limits.
func normalizeTags(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, raw := range in {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			return nil, errors.New("tags must not be blank")
		}
		if len([]rune(tag)) > maxTagLength {
			return nil, fmt.Errorf("tags must be at most %d characters", maxTagLength)
		}
		key := strings.ToLower(tag)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	if len(out) > maxTagsPerTask {
		return nil, fmt.Errorf("a task can have at most %d tags", maxTagsPerTask)
	}
	return out, nil
}

func sortTags(tags []string) {
	sort.SliceStable(tags, func(i, j int) bool { return strings.ToLower(tags[i]) < strings.ToLower(tags[j]) })
}

// replaceTags sets the task's full tag set, creating any tag that doesn't
// exist yet. tags must already be normalized.
func (h *Handler) replaceTags(t *models.Task, tags []string) error {
	if err := h.db.Where("task_id = ?", t.ID).Delete(&models.TaskLabel{}).Error; err != nil {
		return err
	}
	names := make([]string, 0, len(tags))
	for _, tag := range tags {
		var label models.Label
		err := h.db.Where("project_id IS NULL AND LOWER(name) = LOWER(?)", tag).Order("name").Limit(1).Find(&label).Error
		if err != nil {
			return err
		}
		if label.ID == uuid.Nil {
			label = models.Label{Name: tag}
			if err := h.db.Create(&label).Error; err != nil {
				return err
			}
		}
		if err := h.db.Create(&models.TaskLabel{TaskID: t.ID, LabelID: label.ID}).Error; err != nil {
			return err
		}
		names = append(names, label.Name)
	}
	sortTags(names)
	t.Tags = names
	return nil
}

// attachTags loads the tags of the given tasks in one query.
func (h *Handler) attachTags(tasks ...*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	var rows []struct {
		TaskID uuid.UUID
		Name   string
	}
	if err := h.db.Table("task_labels").
		Select("task_labels.task_id, labels.name").
		Joins("JOIN labels ON labels.id = task_labels.label_id").
		Where("task_labels.task_id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return err
	}
	byTask := make(map[uuid.UUID][]string, len(tasks))
	for _, r := range rows {
		byTask[r.TaskID] = append(byTask[r.TaskID], r.Name)
	}
	for _, t := range tasks {
		t.Tags = byTask[t.ID]
		sortTags(t.Tags)
	}
	return nil
}

// ListTags implements GET /api/v1/tags: every tag on at least one task.
func (h *Handler) ListTags(c *gin.Context) {
	var names []string
	if err := h.db.Table("labels").
		Distinct("labels.name").
		Joins("JOIN task_labels ON task_labels.label_id = labels.id").
		Where("labels.project_id IS NULL").
		Pluck("labels.name", &names).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list tags", Details: strptr(err.Error())})
		return
	}
	if names == nil {
		names = []string{}
	}
	sortTags(names)
	c.JSON(http.StatusOK, api.TagList{Data: names})
}
