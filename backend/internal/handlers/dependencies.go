package handlers

import (
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Task dependencies live in task_dependencies: task_id is the blocked task,
// depends_on_id the task that must finish first.

const maxBlockers = 50

func dedupeIDs(in []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]bool, len(in))
	out := make([]uuid.UUID, 0, len(in))
	for _, id := range in {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// validateBlockers checks that every blocker is another task in t's project
// and that blocking t on them wouldn't create a loop.
func (h *Handler) validateBlockers(t *models.Task, blockers []uuid.UUID) error {
	if len(blockers) > maxBlockers {
		return errors.New("a task can wait on at most 50 tasks")
	}
	if len(blockers) == 0 {
		return nil
	}
	for _, id := range blockers {
		if id == t.ID {
			return errors.New("a task can't wait on itself")
		}
	}
	var found []models.Task
	if err := h.db.Where("id IN ?", blockers).Find(&found).Error; err != nil {
		return err
	}
	if len(found) != len(blockers) {
		return errors.New("blocking task not found")
	}
	for _, b := range found {
		if b.ProjectID != t.ProjectID {
			return errors.New("blocking tasks must be in the same project")
		}
	}
	if t.ID == uuid.Nil {
		return nil // a new task can't be anyone's blocker yet
	}
	// Loop check: walk what the blockers wait on; reaching t is a loop.
	var edges []models.TaskDependency
	if err := h.db.Table("task_dependencies AS d").
		Select("d.task_id, d.depends_on_id").
		Joins("JOIN tasks t ON t.id = d.task_id").
		Where("t.project_id = ?", t.ProjectID).
		Scan(&edges).Error; err != nil {
		return err
	}
	waitsOn := map[uuid.UUID][]uuid.UUID{}
	for _, e := range edges {
		if e.TaskID != t.ID { // t's current links are being replaced
			waitsOn[e.TaskID] = append(waitsOn[e.TaskID], e.DependsOnID)
		}
	}
	seen := map[uuid.UUID]bool{}
	stack := append([]uuid.UUID(nil), blockers...)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if id == t.ID {
			return errors.New("these links would make tasks wait on each other in a loop")
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		stack = append(stack, waitsOn[id]...)
	}
	return nil
}

func (h *Handler) replaceBlockers(t *models.Task, blockers []uuid.UUID) error {
	if err := h.db.Where("task_id = ?", t.ID).Delete(&models.TaskDependency{}).Error; err != nil {
		return err
	}
	for _, id := range blockers {
		if err := h.db.Create(&models.TaskDependency{TaskID: t.ID, DependsOnID: id}).Error; err != nil {
			return err
		}
	}
	return h.attachBlockers(t)
}

// attachBlockers loads BlockedByIDs for the tasks in one query, sorted by
// the blocking task's number.
func (h *Handler) attachBlockers(tasks ...*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	var rows []struct {
		TaskID      uuid.UUID
		DependsOnID uuid.UUID
		Number      int
	}
	if err := h.db.Table("task_dependencies AS d").
		Select("d.task_id, d.depends_on_id, b.number").
		Joins("JOIN tasks b ON b.id = d.depends_on_id").
		Where("d.task_id IN ?", ids).
		Scan(&rows).Error; err != nil {
		return err
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Number < rows[j].Number })
	by := map[uuid.UUID][]uuid.UUID{}
	for _, r := range rows {
		by[r.TaskID] = append(by[r.TaskID], r.DependsOnID)
	}
	for _, t := range tasks {
		t.BlockedByIDs = by[t.ID]
	}
	return nil
}
