package handlers

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Task history: activity_log rows with entity_type "task". An update stores
// only the fields that changed, as {"field": {"from": …, "to": …}}.

const entityTask = "task"

func dateOrNil(t interface{ Format(string) string }, isNil bool) any {
	if isNil {
		return nil
	}
	return t.Format("2006-01-02")
}

// taskSnapshot is the task's tracked fields as plain JSON values. The task
// must have its assignees, tags and blockers loaded.
func taskSnapshot(t models.Task) map[string]any {
	str := func(p *string) any {
		if p == nil {
			return nil
		}
		return *p
	}
	id := func(p *uuid.UUID) any {
		if p == nil {
			return nil
		}
		return p.String()
	}
	assignments := map[string]float64{}
	for _, pid := range t.AssigneeIDs {
		assignments[pid.String()] = t.ManDays[pid]
	}
	related := make([]string, len(t.RelatedIDs))
	for i, r := range t.RelatedIDs {
		related[i] = r.String()
	}
	blockers := make([]string, len(t.BlockedByIDs))
	for i, b := range t.BlockedByIDs {
		blockers[i] = b.String()
	}
	tags := append([]string{}, t.Tags...)
	var estimate any
	if t.EstimateManDays != nil {
		estimate = *t.EstimateManDays
	}
	return map[string]any{
		"title":            t.Title,
		"description":      str(t.Description),
		"type":             string(t.Type),
		"status":           string(t.Status),
		"state":            t.State,
		"priority":         string(t.Priority),
		"milestoneId":      id(t.MilestoneID),
		"parentId":         id(t.ParentID),
		"startDate":        dateOrNil(t.StartDate, t.StartDate == nil),
		"dueDate":          dateOrNil(t.DueDate, t.DueDate == nil),
		"estimateManDays":  estimate,
		"storyPoints":      numOrNil(t.StoryPoints),
		"remainingManDays": numOrNil(t.RemainingManDays),
		"completedManDays": numOrNil(t.CompletedManDays),
		"assignments":      assignments,
		"tags":             tags,
		"blockedByIds":     blockers,
		"relatedIds":       related,
		"duplicateOfId":    id(t.DuplicateOfID),
		"sprintId":         id(t.SprintID),
		"areaId":           id(t.AreaID),
		"customValues":     t.Values,
	}
}

func diffSnapshots(before, after map[string]any) map[string]api.TaskFieldChange {
	out := map[string]api.TaskFieldChange{}
	for k, a := range after {
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(before[k])
		if string(ja) != string(jb) {
			out[k] = api.TaskFieldChange{From: before[k], To: a}
		}
	}
	return out
}

// logTask writes a history entry for the task, by the signed-in account.
// History is best effort: a failure here never fails the change itself.
func (h *Handler) logTask(c *gin.Context, action string, taskID uuid.UUID, changes map[string]api.TaskFieldChange) {
	entry := models.ActivityLog{Action: action, EntityType: entityTask, EntityID: &taskID}
	if u, ok := currentUser(c); ok {
		uid := u.ID
		entry.ActorID = &uid
	}
	if len(changes) > 0 {
		b, err := json.Marshal(changes)
		if err == nil {
			s := string(b)
			entry.Changes = &s
		}
	}
	_ = h.db.Create(&entry).Error
}

// GetTaskHistory implements GET /api/v1/tasks/{id}/history.
func (h *Handler) GetTaskHistory(c *gin.Context) {
	t, ok := h.loadTask(c)
	if !ok {
		return
	}
	var rows []struct {
		models.ActivityLog
		PersonID *uuid.UUID
	}
	if err := h.db.Table("activity_log AS a").
		Select("a.*, u.person_id").
		Joins("LEFT JOIN users u ON u.id = a.actor_id").
		Where("a.entity_type = ? AND a.entity_id = ?", entityTask, t.ID).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load history", Details: strptr(err.Error())})
		return
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].CreatedAt.After(rows[j].CreatedAt) })
	out := make([]api.TaskHistoryEntry, len(rows))
	for i, r := range rows {
		changes := map[string]api.TaskFieldChange{}
		if r.Changes != nil {
			_ = json.Unmarshal([]byte(*r.Changes), &changes)
		}
		out[i] = api.TaskHistoryEntry{
			Id: r.ID, Action: api.TaskHistoryEntryAction(r.Action), ActorPersonId: r.PersonID,
			Changes: changes, CreatedAt: r.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, api.TaskHistory{Data: out})
}

func numOrNil(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}
