package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Built-in fields rules can name, with their labels.
var builtInFields = map[string]string{
	"assignees": "Assignees", "dueDate": "Due date", "startDate": "Start date", "storyPoints": "Story points",
	"manDays": "Man-days", "description": "Description", "area": "Area", "sprint": "Sprint", "tags": "Tags",
}

// validateRules checks rule names against the project's states and fields.
func validateRules(r api.WorkItemRules, states []api.WorkflowState, fields []api.CustomField) error {
	stateOK := map[string]bool{}
	for _, s := range states {
		stateOK[s.Key] = true
	}
	field := map[string]api.CustomField{}
	for _, f := range fields {
		field[f.Key] = f
	}
	for st, names := range r.RequiredByState {
		if !stateOK[st] {
			return fmt.Errorf("this project has no state %q", st)
		}
		for _, n := range names {
			if _, ok := builtInFields[n]; !ok {
				if _, ok := field[n]; !ok {
					return fmt.Errorf("unknown field %q", n)
				}
			}
		}
	}
	for k, v := range r.Defaults {
		switch k {
		case "priority":
			if s, ok := v.(string); !ok || !api.TaskPriority(s).Valid() {
				return fmt.Errorf("unknown default priority %v", v)
			}
		case "storyPoints":
			if n, ok := v.(float64); !ok || n < 0 || n > 999 {
				return fmt.Errorf("default story points must be 0–999")
			}
		default:
			if _, ok := field[k]; !ok {
				return fmt.Errorf("unknown field %q for a default", k)
			}
			if _, err := checkCustomValues(fieldsWithout(fields), map[string]any{k: v}); err != nil {
				return fmt.Errorf("default for %s: %v", k, err)
			}
		}
	}
	for _, k := range r.ReadOnly {
		if _, ok := field[k]; !ok {
			return fmt.Errorf("only custom fields can be read-only, not %q", k)
		}
	}
	return nil
}

// fieldsWithout drops "required" so a single value can be checked alone.
func fieldsWithout(fields []api.CustomField) []api.CustomField {
	out := make([]api.CustomField, len(fields))
	for i, f := range fields {
		f.Required = false
		out[i] = f
	}
	return out
}

func (h *Handler) projectRules(projectID uuid.UUID) (api.WorkItemRules, []api.CustomField) {
	var p models.Project
	if h.db.Select("id, rules, custom_fields").First(&p, "id = ?", projectID).Error != nil {
		return api.WorkItemRules{}, nil
	}
	return p.RulesOf(), p.Fields()
}

// applyDefaults fills a new item's unset priority, points and custom fields.
func applyDefaults(t *models.Task, in api.TaskCreate, values map[string]any, r api.WorkItemRules) {
	for k, v := range r.Defaults {
		switch k {
		case "priority":
			if in.Priority == nil {
				t.Priority = api.TaskPriority(v.(string))
			}
		case "storyPoints":
			if in.StoryPoints == nil {
				n := v.(float64)
				t.StoryPoints = &n
			}
		default:
			if _, set := values[k]; !set {
				values[k] = v
			}
		}
	}
}

// itemFields says which fields an item has filled, for required checks.
type itemFields struct {
	assignees, tags int
	t               *models.Task
}

func (f itemFields) has(name string) bool {
	t := f.t
	switch name {
	case "assignees":
		return f.assignees > 0
	case "tags":
		return f.tags > 0
	case "dueDate":
		return t.DueDate != nil
	case "startDate":
		return t.StartDate != nil
	case "storyPoints":
		return t.StoryPoints != nil
	case "manDays":
		return t.EstimateManDays != nil && *t.EstimateManDays > 0
	case "description":
		return t.Description != nil && strings.TrimSpace(*t.Description) != ""
	case "area":
		return t.AreaID != nil
	case "sprint":
		return t.SprintID != nil
	}
	v, ok := t.Values[name]
	return ok && v != nil && v != ""
}

// checkRequired answers 400 when the item's state needs fields it lacks.
func (h *Handler) checkRequired(c *gin.Context, f itemFields) bool {
	r, fields := h.projectRules(f.t.ProjectID)
	names := r.RequiredByState[f.t.State]
	if len(names) == 0 {
		return true
	}
	label := map[string]string{}
	for k, v := range builtInFields {
		label[k] = v
	}
	for _, cf := range fields {
		label[cf.Key] = cf.Label
	}
	var missing []string
	for _, n := range names {
		if !f.has(n) {
			missing = append(missing, label[n])
		}
	}
	if len(missing) == 0 {
		return true
	}
	sort.Strings(missing)
	c.JSON(http.StatusBadRequest, api.Error{Message: fmt.Sprintf("%s needs: %s", h.stateLabel(f.t), strings.Join(missing, ", "))})
	return false
}

func (h *Handler) stateLabel(t *models.Task) string {
	for _, s := range h.projectStates(t) {
		if s.Key == t.State {
			return s.Label
		}
	}
	return t.State
}

// checkReadOnly answers 403 when a member changes a read-only field.
func (h *Handler) checkReadOnly(c *gin.Context, projectID uuid.UUID, before, after map[string]any) bool {
	if h.hasRole(c, api.SystemRoleManager) {
		return true
	}
	r, fields := h.projectRules(projectID)
	for _, k := range r.ReadOnly {
		if fmt.Sprint(before[k]) != fmt.Sprint(after[k]) {
			name := k
			for _, f := range fields {
				if f.Key == k {
					name = f.Label
				}
			}
			c.JSON(http.StatusForbidden, api.Error{Message: fmt.Sprintf("%s can only be changed by managers and admins", name)})
			return false
		}
	}
	return true
}
