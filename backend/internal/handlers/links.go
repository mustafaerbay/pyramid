package handlers

import (
	"errors"
	"sort"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// "Related" links go both ways and are stored once per pair, with the
// smaller id first (task_related). "Duplicate of" is tasks.duplicate_of_id.

type relatedPair struct {
	TaskID  uuid.UUID `gorm:"type:uuid;primaryKey"`
	OtherID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (relatedPair) TableName() string { return "task_related" }

func orderedPair(a, b uuid.UUID) relatedPair {
	if a.String() < b.String() {
		return relatedPair{TaskID: a, OtherID: b}
	}
	return relatedPair{TaskID: b, OtherID: a}
}

// sameProject reports whether every id is a task in project (and not t).
func (h *Handler) sameProject(t *models.Task, ids []uuid.UUID, what string) error {
	for _, id := range ids {
		if id == t.ID {
			return errors.New("a task can't be " + what + " itself")
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var n int64
	if err := h.db.Model(&models.Task{}).Where("id IN ? AND project_id = ?", ids, t.ProjectID).Count(&n).Error; err != nil {
		return err
	}
	if int(n) != len(ids) {
		return errors.New("linked tasks must exist in the same project")
	}
	return nil
}

func (h *Handler) validateLinks(t *models.Task, related []uuid.UUID) error {
	if len(related) > 50 {
		return errors.New("a task can have at most 50 related links")
	}
	if err := h.sameProject(t, related, "related to"); err != nil {
		return err
	}
	if t.DuplicateOfID == nil {
		return nil
	}
	if err := h.sameProject(t, []uuid.UUID{*t.DuplicateOfID}, "a duplicate of"); err != nil {
		return err
	}
	var target models.Task
	if err := h.db.First(&target, "id = ?", *t.DuplicateOfID).Error; err != nil {
		return err
	}
	if target.DuplicateOfID != nil && *target.DuplicateOfID == t.ID {
		return errors.New("that task is already marked as a duplicate of this one")
	}
	return nil
}

func (h *Handler) replaceRelated(t *models.Task, ids []uuid.UUID) error {
	if err := h.db.Where("task_id = ? OR other_id = ?", t.ID, t.ID).Delete(&relatedPair{}).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := h.db.Create(ptr(orderedPair(t.ID, id))).Error; err != nil {
			return err
		}
	}
	return h.attachRelated(t)
}

func ptr[T any](v T) *T { return &v }

// attachRelated loads RelatedIDs (both directions), sorted by task number.
func (h *Handler) attachRelated(tasks ...*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	var pairs []relatedPair
	if err := h.db.Where("task_id IN ? OR other_id IN ?", ids, ids).Find(&pairs).Error; err != nil {
		return err
	}
	by := map[uuid.UUID][]uuid.UUID{}
	others := map[uuid.UUID]bool{}
	for _, p := range pairs {
		by[p.TaskID] = append(by[p.TaskID], p.OtherID)
		by[p.OtherID] = append(by[p.OtherID], p.TaskID)
		others[p.TaskID], others[p.OtherID] = true, true
	}
	number := map[uuid.UUID]int{}
	if len(others) > 0 {
		list := make([]uuid.UUID, 0, len(others))
		for id := range others {
			list = append(list, id)
		}
		var rows []struct {
			ID     uuid.UUID
			Number int
		}
		if err := h.db.Model(&models.Task{}).Select("id, number").Where("id IN ?", list).Scan(&rows).Error; err != nil {
			return err
		}
		for _, r := range rows {
			number[r.ID] = r.Number
		}
	}
	for _, t := range tasks {
		rel := by[t.ID]
		sort.Slice(rel, func(i, j int) bool { return number[rel[i]] < number[rel[j]] })
		t.RelatedIDs = rel
	}
	return nil
}
