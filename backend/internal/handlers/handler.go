package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/config"
	"gorm.io/gorm"
)

// Handler carries shared dependencies for all HTTP handlers.
type Handler struct {
	db     *gorm.DB
	cfg    config.Config
	logins *loginLimiter
	// mail sends one notification email; nil when email is off.
	mail func(to, subject, body string) error
}

// New builds a Handler.
func New(db *gorm.DB, cfg config.Config) *Handler {
	return &Handler{db: db, cfg: cfg, logins: newLoginLimiter(), mail: smtpSender(cfg)}
}

// RegisterRoutes wires every route. Route paths mirror openapi/openapi.yaml.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.Use(h.cors())

	r.GET("/health", h.GetHealth)

	// Signing in (and resetting a forgotten password) need no session.
	r.POST("/api/v1/auth/login", h.Login)
	r.GET("/api/v1/auth/options", h.GetAuthOptions)
	r.POST("/api/v1/auth/forgot", h.ForgotPassword)
	r.POST("/api/v1/auth/reset", h.ResetPassword)
	r.POST("/api/v1/git/hooks/:id", h.ReceiveGitHook)

	v1 := r.Group("/api/v1", h.requireAuth(), h.denyViewerWrites(), h.auditTrail())
	manager := h.requireRole(api.SystemRoleManager)
	admin := h.requireRole(api.SystemRoleAdmin)
	{
		v1.POST("/auth/logout", h.Logout)
		v1.GET("/auth/me", h.GetMe)
		v1.POST("/auth/password", h.ChangePassword)
		v1.GET("/notifications", h.ListNotifications)
		v1.POST("/notifications/read-all", h.MarkAllNotificationsRead)
		v1.POST("/notifications/:id/read", h.MarkNotificationRead)
		v1.GET("/users", admin, h.ListUsers)
		v1.POST("/users", admin, h.CreateUser)
		v1.PATCH("/users/:id", admin, h.UpdateUser)

		v1.GET("/people", h.ListPeople)
		v1.POST("/people", manager, h.CreatePerson)
		v1.GET("/people/:id", h.GetPerson)
		v1.PATCH("/people/:id", manager, h.UpdatePerson)
		v1.DELETE("/people/:id", manager, h.DeletePerson)

		v1.GET("/projects", h.ListProjects)
		v1.POST("/projects", h.CreateProject)
		v1.GET("/projects/:id", h.GetProject)
		v1.PATCH("/projects/:id", h.UpdateProject)
		v1.DELETE("/projects/:id", h.DeleteProject)
		v1.GET("/projects/:id/flow", h.GetProjectFlow)

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

		v1.GET("/weekly-summaries", h.ListWeeklySummaries)
		v1.POST("/weekly-summaries", h.CreateWeeklySummary)
		v1.GET("/weekly-summaries/:id", h.GetWeeklySummary)
		v1.PATCH("/weekly-summaries/:id", h.UpdateWeeklySummary)
		v1.DELETE("/weekly-summaries/:id", h.DeleteWeeklySummary)

		v1.GET("/versions", h.ListVersions)
		v1.POST("/versions", h.CreateVersion)
		v1.GET("/versions/:id", h.GetVersion)
		v1.PATCH("/versions/:id", h.UpdateVersion)
		v1.DELETE("/versions/:id", h.DeleteVersion)

		v1.POST("/queries/run", h.RunQuery)
		v1.GET("/search", h.Search)
		v1.GET("/delivery-plans", h.ListDeliveryPlans)
		v1.POST("/delivery-plans", h.CreateDeliveryPlan)
		v1.PATCH("/delivery-plans/:id", h.UpdateDeliveryPlan)
		v1.DELETE("/delivery-plans/:id", h.DeleteDeliveryPlan)
		v1.GET("/project-hooks", manager, h.ListProjectHooks)
		v1.POST("/project-hooks", manager, h.CreateProjectHook)
		v1.PATCH("/project-hooks/:id", manager, h.UpdateProjectHook)
		v1.DELETE("/project-hooks/:id", manager, h.DeleteProjectHook)
		v1.POST("/project-hooks/:id/test", manager, h.TestProjectHook)
		v1.GET("/project-packages", h.ListProjectPackages)
		v1.POST("/project-packages", manager, h.CreateProjectPackage)
		v1.DELETE("/project-packages/:id", manager, h.DeleteProjectPackage)
		v1.GET("/project-packages/:id/versions", h.ListPackageVersions)
		v1.PUT("/project-packages/:id/links", h.LinkArtifactVersion)
		v1.GET("/wiki-pages", h.ListWikiPages)
		v1.POST("/wiki-pages", h.CreateWikiPage)
		v1.GET("/wiki-pages/:id", h.GetWikiPage)
		v1.PATCH("/wiki-pages/:id", h.UpdateWikiPage)
		v1.DELETE("/wiki-pages/:id", h.DeleteWikiPage)
		v1.GET("/wiki-pages/:id/revisions", h.ListWikiRevisions)
		v1.GET("/project-repos", h.ListProjectRepos)
		v1.POST("/project-repos", manager, h.CreateProjectRepo)
		v1.DELETE("/project-repos/:id", manager, h.DeleteProjectRepo)
		v1.GET("/git-activity", h.ListGitActivity)
		v1.GET("/me/tokens", h.ListApiTokens)
		v1.POST("/me/tokens", h.CreateApiToken)
		v1.DELETE("/me/tokens/:id", h.RevokeApiToken)
		v1.GET("/audit", admin, h.ListAudit)
		v1.GET("/me/notification-settings", h.GetNotificationSettings)
		v1.PUT("/me/notification-settings", h.PutNotificationSettings)
		v1.POST("/tasks/:id/transfer", h.TransferTask)
		v1.GET("/item-templates", h.ListItemTemplates)
		v1.POST("/item-templates", h.CreateItemTemplate)
		v1.PATCH("/item-templates/:id", h.UpdateItemTemplate)
		v1.DELETE("/item-templates/:id", h.DeleteItemTemplate)
		v1.GET("/tasks/:id/followers", h.GetTaskFollowers)
		v1.PUT("/tasks/:id/follow", h.FollowTask)
		v1.DELETE("/tasks/:id/follow", h.UnfollowTask)
		v1.GET("/dashboards", h.ListDashboards)
		v1.POST("/dashboards", h.CreateDashboard)
		v1.GET("/dashboards/:id", h.GetDashboard)
		v1.PATCH("/dashboards/:id", h.UpdateDashboard)
		v1.DELETE("/dashboards/:id", h.DeleteDashboard)
		v1.GET("/queries", h.ListSavedQueries)
		v1.POST("/queries", h.CreateSavedQuery)
		v1.PATCH("/queries/:id", h.UpdateSavedQuery)
		v1.DELETE("/queries/:id", h.DeleteSavedQuery)

		v1.GET("/process-templates", h.ListProcessTemplates)
		v1.POST("/process-templates", manager, h.CreateProcessTemplate)
		v1.DELETE("/process-templates/:id", manager, h.DeleteProcessTemplate)
		v1.POST("/projects/:id/apply-process", h.ApplyProcess)

		v1.GET("/project-teams", h.ListProjectTeams)
		v1.POST("/project-teams", h.CreateProjectTeam)
		v1.PATCH("/project-teams/:id", h.UpdateProjectTeam)
		v1.DELETE("/project-teams/:id", h.DeleteProjectTeam)

		v1.GET("/areas", h.ListAreas)
		v1.POST("/areas", h.CreateArea)
		v1.PATCH("/areas/:id", h.UpdateArea)
		v1.DELETE("/areas/:id", h.DeleteArea)

		v1.GET("/sprints", h.ListSprints)
		v1.POST("/sprints", h.CreateSprint)
		v1.GET("/sprints/:id", h.GetSprint)
		v1.PATCH("/sprints/:id", h.UpdateSprint)
		v1.DELETE("/sprints/:id", h.DeleteSprint)
		v1.GET("/sprints/:id/report", h.GetSprintReport)
		v1.PUT("/sprints/:id/capacity", h.SetSprintCapacity)

		v1.GET("/capacity", h.GetCapacity)
		v1.GET("/people/:id/workload", h.GetPersonWorkload)
		v1.GET("/tags", h.ListTags)
		v1.POST("/tasks/:id/move", h.MoveTask)
		v1.GET("/tasks/:id/comments", h.ListTaskComments)
		v1.POST("/tasks/:id/comments", h.CreateTaskComment)
		v1.PATCH("/task-comments/:id", h.UpdateTaskComment)
		v1.DELETE("/task-comments/:id", h.DeleteTaskComment)
		v1.GET("/tasks/:id/history", h.GetTaskHistory)
		v1.GET("/tasks/:id/attachments", h.ListTaskAttachments)
		v1.POST("/tasks/:id/attachments", h.UploadTaskAttachment)
		v1.GET("/attachments/:id/download", h.DownloadAttachment)
		v1.DELETE("/attachments/:id", h.DeleteAttachment)
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
		// The session cookie only travels cross-origin (local dev: :3000 →
		// :8080) when credentials are allowed for that exact origin.
		if h.cfg.CORSOrigin != "*" {
			c.Header("Access-Control-Allow-Credentials", "true")
		}
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
