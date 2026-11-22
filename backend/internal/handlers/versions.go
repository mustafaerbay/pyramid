package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

const versionNameTakenMsg = "a version with this name already exists in this project"

// errVersionNameTaken signals a 409 out of a transaction callback.
var errVersionNameTaken = errors.New(versionNameTakenMsg)

// ListVersions implements GET /api/v1/versions. Filter by projectId for one
// project's release plan. Ordered by target date (undated last), then name.
func (h *Handler) ListVersions(c *gin.Context) {
	page := parseIntDefault(c.Query("page"), 1)
	if page < 1 {
		page = 1
	}
	pageSize := clamp(parseIntDefault(c.Query("pageSize"), 25), 1, 100)

	q := h.db.Model(&models.Version{})
	if p := c.Query("projectId"); p != "" {
		q = q.Where("project_id = ?", p)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to count versions", Details: strptr(err.Error())})
		return
	}

	var versions []models.Version
	// "target_date IS NULL" sorts false (dated) before true (undated) on
	// both Postgres and SQLite — a portable NULLS LAST.
	if err := q.Order("target_date IS NULL, target_date ASC, name ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list versions", Details: strptr(err.Error())})
		return
	}

	data := make([]api.Version, len(versions))
	for i := range versions {
		data[i] = versions[i].ToAPI()
	}
	c.JSON(http.StatusOK, api.VersionList{
		Data:       data,
		Pagination: api.Pagination{Page: page, PageSize: pageSize, Total: total},
	})
}

// versionNameTaken reports whether another version of the project already
// uses name. excludeID skips the version being renamed.
func versionNameTaken(tx *gorm.DB, projectID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	q := tx.Model(&models.Version{}).Where("project_id = ? AND name = ?", projectID, name)
	if excludeID != nil {
		q = q.Where("id <> ?", *excludeID)
	}
	var n int64
	if err := q.Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// CreateVersion implements POST /api/v1/versions.
func (h *Handler) CreateVersion(c *gin.Context) {
	var in api.VersionCreate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	v := models.NewVersionFromCreate(in)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		taken, err := versionNameTaken(tx, v.ProjectID, v.Name, nil)
		if err != nil {
			return err
		}
		if taken {
			return errVersionNameTaken
		}
		return tx.Create(&v).Error
	})
	if errors.Is(err, errVersionNameTaken) {
		c.JSON(http.StatusConflict, api.Error{Message: versionNameTakenMsg})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to create version", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, v.ToAPI())
}

// loadVersion fetches a version by the :id param, writing the 404/500
// response itself when it can't. ok is false when the caller should return.
func (h *Handler) loadVersion(c *gin.Context) (v models.Version, ok bool) {
	if err := h.db.First(&v, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "version not found"})
			return v, false
		}
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load version", Details: strptr(err.Error())})
		return v, false
	}
	return v, true
}

// GetVersion implements GET /api/v1/versions/{id}.
func (h *Handler) GetVersion(c *gin.Context) {
	v, ok := h.loadVersion(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, v.ToAPI())
}

// UpdateVersion implements PATCH /api/v1/versions/{id}.
func (h *Handler) UpdateVersion(c *gin.Context) {
	v, ok := h.loadVersion(c)
	if !ok {
		return
	}

	var in api.VersionUpdate
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}

	v.ApplyUpdate(in)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if in.Name != nil {
			taken, err := versionNameTaken(tx, v.ProjectID, v.Name, &v.ID)
			if err != nil {
				return err
			}
			if taken {
				return errVersionNameTaken
			}
		}
		return tx.Save(&v).Error
	})
	if errors.Is(err, errVersionNameTaken) {
		c.JSON(http.StatusConflict, api.Error{Message: versionNameTakenMsg})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to update version", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusOK, v.ToAPI())
}

// DeleteVersion implements DELETE /api/v1/versions/{id}. Its milestones are
// kept and unassigned first, in the same transaction; the DB's
// ON DELETE SET NULL is only the backstop.
func (h *Handler) DeleteVersion(c *gin.Context) {
	id := c.Param("id")
	var deleted int64
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Milestone{}).Where("version_id = ?", id).Update("version_id", nil).Error; err != nil {
			return err
		}
		res := tx.Delete(&models.Version{}, "id = ?", id)
		deleted = res.RowsAffected
		return res.Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete version", Details: strptr(err.Error())})
		return
	}
	if deleted == 0 {
		c.JSON(http.StatusNotFound, api.Error{Message: "version not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
