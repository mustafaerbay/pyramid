package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
	"gorm.io/gorm"
)

// Teams inside a project (project_teams + project_team_members). A team
// owns area paths; items in those areas (and below) are the team's.

type projectTeam struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null"`
	Name      string    `gorm:"not null"`
	AreaIDs   string    `gorm:"column:area_ids;type:jsonb;not null"`
	CreatedAt time.Time
}

func (projectTeam) TableName() string { return "project_teams" }

type teamMember struct {
	TeamID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	PersonID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (teamMember) TableName() string { return "project_team_members" }

func (h *Handler) teamJSON(t projectTeam) api.ProjectTeam {
	var areas []uuid.UUID
	_ = json.Unmarshal([]byte(t.AreaIDs), &areas)
	var members []uuid.UUID
	h.db.Model(&teamMember{}).Where("team_id = ?", t.ID).Pluck("person_id", &members)
	if members == nil {
		members = []uuid.UUID{}
	}
	if areas == nil {
		areas = []uuid.UUID{}
	}
	return api.ProjectTeam{Id: t.ID, ProjectId: t.ProjectID, Name: t.Name, MemberIds: members, AreaIds: areas}
}

// ListProjectTeams implements GET /api/v1/project-teams.
func (h *Handler) ListProjectTeams(c *gin.Context) {
	pid, err := uuid.Parse(c.Query("projectId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "projectId is required"})
		return
	}
	var rows []projectTeam
	h.db.Where("project_id = ?", pid).Order("name").Find(&rows)
	out := make([]api.ProjectTeam, len(rows))
	for i, t := range rows {
		out[i] = h.teamJSON(t)
	}
	c.JSON(http.StatusOK, api.ProjectTeamList{Data: out})
}

// saveTeam validates input and writes the team with its members.
func (h *Handler) saveTeam(c *gin.Context, t projectTeam, in api.ProjectTeamInput, status int) {
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 60 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a team needs a name of 1–60 characters"})
		return
	}
	var dup int64
	h.db.Model(&projectTeam{}).Where("project_id = ? AND LOWER(name) = LOWER(?) AND id <> ?", t.ProjectID, name, t.ID).Count(&dup)
	if dup > 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "this project already has a team with that name"})
		return
	}
	areas := []uuid.UUID{}
	if in.AreaIds != nil {
		areas = dedupeIDs(*in.AreaIds)
		var n int64
		h.db.Model(&area{}).Where("id IN ? AND project_id = ?", areas, t.ProjectID).Count(&n)
		if len(areas) > 0 && int(n) != len(areas) {
			c.JSON(http.StatusBadRequest, api.Error{Message: "a team's areas must be this project's"})
			return
		}
	}
	b, _ := json.Marshal(areas)
	t.Name, t.AreaIDs = name, string(b)
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&t).Error; err != nil {
			return err
		}
		if in.MemberIds == nil {
			return nil
		}
		if err := tx.Where("team_id = ?", t.ID).Delete(&teamMember{}).Error; err != nil {
			return err
		}
		for _, p := range dedupeIDs(*in.MemberIds) {
			if err := tx.Create(&teamMember{TeamID: t.ID, PersonID: p}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to save team", Details: strptr(err.Error())})
		return
	}
	c.JSON(status, h.teamJSON(t))
}

// CreateProjectTeam implements POST /api/v1/project-teams.
func (h *Handler) CreateProjectTeam(c *gin.Context) {
	var in api.ProjectTeamInput
	if err := c.ShouldBindJSON(&in); err != nil || in.ProjectId == nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a team needs a projectId and a name"})
		return
	}
	if !h.canWriteProject(c, *in.ProjectId) {
		return
	}
	h.saveTeam(c, projectTeam{ID: uuid.New(), ProjectID: *in.ProjectId, CreatedAt: time.Now()}, in, http.StatusCreated)
}

func (h *Handler) loadTeam(c *gin.Context) (projectTeam, bool) {
	var t projectTeam
	if err := h.db.First(&t, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "team not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load team"})
		}
		return t, false
	}
	return t, true
}

// UpdateProjectTeam implements PATCH /api/v1/project-teams/{id}.
func (h *Handler) UpdateProjectTeam(c *gin.Context) {
	t, ok := h.loadTeam(c)
	if !ok || !h.canWriteProject(c, t.ProjectID) {
		return
	}
	var in api.ProjectTeamInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body"})
		return
	}
	h.saveTeam(c, t, in, http.StatusOK)
}

// DeleteProjectTeam implements DELETE /api/v1/project-teams/{id}.
func (h *Handler) DeleteProjectTeam(c *gin.Context) {
	t, ok := h.loadTeam(c)
	if !ok || !h.canWriteProject(c, t.ProjectID) {
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Sprint{}).Where("team_id = ?", t.ID).Update("team_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Where("team_id = ?", t.ID).Delete(&teamMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(&t).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete team"})
		return
	}
	c.Status(http.StatusNoContent)
}

// validateSprintTeam checks the sprint's team is in its project.
func (h *Handler) validateSprintTeam(s models.Sprint) error {
	if s.TeamID == nil {
		return nil
	}
	var n int64
	h.db.Model(&projectTeam{}).Where("id = ? AND project_id = ?", *s.TeamID, s.ProjectID).Count(&n)
	if n == 0 {
		return errors.New("the team must belong to the sprint's project")
	}
	return nil
}

// canWriteProject answers 403 (and false) when the signed-in account may
// not change the project: a restricted project accepts only its members
// with write access, managers and admins.
func (h *Handler) canWriteProject(c *gin.Context, projectID uuid.UUID) bool {
	if h.cfg.AuthOff || h.hasRole(c, api.SystemRoleManager) {
		return true
	}
	var p models.Project
	if err := h.db.Select("id, restricted").First(&p, "id = ?", projectID).Error; err != nil || !p.Restricted {
		return true // unknown project: the handler reports it
	}
	u, _ := currentUser(c)
	var n int64
	h.db.Model(&models.ProjectMember{}).Where("project_id = ? AND person_id = ? AND access = ?", projectID, u.PersonID, "write").Count(&n)
	if n == 0 {
		c.JSON(http.StatusForbidden, api.Error{Message: "this project is restricted to its members"})
		return false
	}
	return true
}

// canWriteTask is canWriteProject for the project of the task in :id.
func (h *Handler) canWriteTask(c *gin.Context) bool {
	var t models.Task
	if err := h.db.Select("id, project_id").First(&t, "id = ?", c.Param("id")).Error; err != nil {
		return true
	}
	return h.canWriteProject(c, t.ProjectID)
}
