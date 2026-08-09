package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"gorm.io/gorm"
)

// Handler carries shared dependencies for all HTTP handlers.
type Handler struct {
	db  *gorm.DB
	cfg config.Config
}

// New builds a Handler.
func New(db *gorm.DB, cfg config.Config) *Handler {
	return &Handler{db: db, cfg: cfg}
}

// RegisterRoutes wires every route. Route paths mirror openapi/openapi.yaml.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.Use(h.cors())

	r.GET("/health", h.GetHealth)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/people", h.ListPeople)
		v1.POST("/people", h.CreatePerson)
		v1.GET("/people/:id", h.GetPerson)
		v1.PATCH("/people/:id", h.UpdatePerson)
		v1.DELETE("/people/:id", h.DeletePerson)

		v1.GET("/projects", h.ListProjects)
		v1.POST("/projects", h.CreateProject)
		v1.GET("/projects/:id", h.GetProject)
		v1.PATCH("/projects/:id", h.UpdateProject)
		v1.DELETE("/projects/:id", h.DeleteProject)

		v1.GET("/project-members", h.ListProjectMembers)
		v1.POST("/project-members", h.CreateProjectMember)
		v1.GET("/project-members/:id", h.GetProjectMember)
		v1.PATCH("/project-members/:id", h.UpdateProjectMember)
		v1.DELETE("/project-members/:id", h.DeleteProjectMember)

		v1.GET("/project-kickoffs", h.ListProjectKickoffs)
		v1.POST("/project-kickoffs", h.CreateProjectKickoff)
		v1.GET("/project-kickoffs/:id", h.GetProjectKickoff)
		v1.PATCH("/project-kickoffs/:id", h.UpdateProjectKickoff)
		v1.DELETE("/project-kickoffs/:id", h.DeleteProjectKickoff)

		v1.GET("/milestones", h.ListMilestones)
		v1.POST("/milestones", h.CreateMilestone)
		v1.GET("/milestones/:id", h.GetMilestone)
		v1.PATCH("/milestones/:id", h.UpdateMilestone)
		v1.DELETE("/milestones/:id", h.DeleteMilestone)

		v1.GET("/tasks", h.ListTasks)
		v1.POST("/tasks", h.CreateTask)
		v1.GET("/tasks/:id", h.GetTask)
		v1.PATCH("/tasks/:id", h.UpdateTask)
		v1.DELETE("/tasks/:id", h.DeleteTask)

		v1.GET("/goals", h.ListGoals)
		v1.POST("/goals", h.CreateGoal)
		v1.GET("/goals/:id", h.GetGoal)
		v1.PATCH("/goals/:id", h.UpdateGoal)
		v1.DELETE("/goals/:id", h.DeleteGoal)

		v1.GET("/leave-requests", h.ListLeaveRequests)
		v1.POST("/leave-requests", h.CreateLeaveRequest)
		v1.GET("/leave-requests/:id", h.GetLeaveRequest)
		v1.PATCH("/leave-requests/:id", h.UpdateLeaveRequest)
		v1.DELETE("/leave-requests/:id", h.DeleteLeaveRequest)

		v1.GET("/contributions", h.ListContributions)
		v1.POST("/contributions", h.CreateContribution)
		v1.GET("/contributions/:id", h.GetContribution)
		v1.PATCH("/contributions/:id", h.UpdateContribution)
		v1.DELETE("/contributions/:id", h.DeleteContribution)
	}
}

// cors is a minimal CORS middleware for the local Svelte dev server.
func (h *Handler) cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", h.cfg.CORSOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// --- small shared helpers ---

func strptr(s string) *string { return &s }

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
