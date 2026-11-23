package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Pyramid doesn't host Git or run builds. A project links its GitHub /
// GitLab repositories; their webhooks report commits, pull requests and
// pipeline runs, which are linked to the items they mention (KEY-12, #12).

type projectRepo struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null"`
	Provider    string    `gorm:"not null"`
	Name        string    `gorm:"not null"`
	URL         string    `gorm:"column:url;not null"`
	Secret      string    `gorm:"not null"`
	LastEventAt *time.Time
	CreatedAt   time.Time
}

func (projectRepo) TableName() string { return "project_repos" }

func (r projectRepo) toAPI(withSecret bool) api.ProjectRepo {
	out := api.ProjectRepo{Id: r.ID, ProjectId: r.ProjectID, Provider: api.ProjectRepoProvider(r.Provider), Name: r.Name, Url: r.URL, LastEventAt: r.LastEventAt, CreatedAt: r.CreatedAt}
	if withSecret {
		s := r.Secret
		out.Secret = &s
	}
	return out
}

type gitEvent struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	RepoID     uuid.UUID `gorm:"type:uuid;not null"`
	ProjectID  uuid.UUID `gorm:"type:uuid;not null"`
	Kind       string    `gorm:"not null"`
	ExternalID string    `gorm:"not null"`
	Title      string    `gorm:"not null"`
	URL        *string   `gorm:"column:url"`
	Author     *string
	State      *string
	Ref        *string
	Sha        *string
	Items      string `gorm:"type:jsonb;not null"`
	OccurredAt time.Time
	CreatedAt  time.Time
}

func (gitEvent) TableName() string { return "git_events" }

func (e gitEvent) numbers() []int {
	var n []int
	_ = json.Unmarshal([]byte(e.Items), &n)
	return n
}

// itemNumbers finds KEY-12 and #12 in texts.
func itemNumbers(key string, texts ...string) []int {
	re := regexp.MustCompile(`(?i)(?:\b` + regexp.QuoteMeta(key) + `-|(?:^|[^\w&])#)(\d{1,7})\b`)
	seen := map[int]bool{}
	var out []int
	for _, t := range texts {
		for _, m := range re.FindAllStringSubmatch(t, -1) {
			n, _ := strconv.Atoi(m[1])
			if n > 0 && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Ints(out)
	return out
}

var repoPath = regexp.MustCompile(`^/([^/]+(?:/[^/]+)+?)(?:\.git)?/?$`)

// CreateProjectRepo implements POST /api/v1/project-repos.
func (h *Handler) CreateProjectRepo(c *gin.Context) {
	var in api.ProjectRepoInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "invalid request body", Details: strptr(err.Error())})
		return
	}
	var n int64
	h.db.Model(&models.Project{}).Where("id = ?", in.ProjectId).Count(&n)
	if n == 0 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "unknown project"})
		return
	}
	if !in.Provider.Valid() {
		c.JSON(http.StatusBadRequest, api.Error{Message: "provider is github or gitlab"})
		return
	}
	u, err := url.Parse(strings.TrimSpace(in.Url))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" || len(in.Url) > 500 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "give the repository's web address, e.g. https://github.com/acme/shop"})
		return
	}
	m := repoPath.FindStringSubmatch(u.Path)
	if m == nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "that address has no owner/repository path"})
		return
	}
	buf := make([]byte, 20)
	_, _ = rand.Read(buf)
	r := projectRepo{ProjectID: in.ProjectId, Provider: string(in.Provider), Name: m[1], URL: strings.TrimSuffix(strings.TrimSuffix(u.String(), "/"), ".git"), Secret: hex.EncodeToString(buf)}
	if err := h.db.Create(&r).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "failed to link the repository", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, r.toAPI(true))
}

// ListProjectRepos implements GET /api/v1/project-repos.
func (h *Handler) ListProjectRepos(c *gin.Context) {
	var rows []projectRepo
	h.db.Where("project_id = ?", c.Query("projectId")).Order("name").Find(&rows)
	out := make([]api.ProjectRepo, len(rows))
	for i, r := range rows {
		out[i] = r.toAPI(false)
	}
	c.JSON(http.StatusOK, api.ProjectRepoList{Data: out})
}

// DeleteProjectRepo implements DELETE /api/v1/project-repos/{id}.
func (h *Handler) DeleteProjectRepo(c *gin.Context) {
	var r projectRepo
	if err := h.db.First(&r, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "repository not found"})
		return
	}
	h.db.Where("repo_id = ?", r.ID).Delete(&gitEvent{})
	h.db.Delete(&r)
	c.Status(http.StatusNoContent)
}

// ListGitActivity implements GET /api/v1/git-activity.
func (h *Handler) ListGitActivity(c *gin.Context) {
	q := h.db.Model(&gitEvent{})
	number := 0
	switch {
	case c.Query("taskId") != "":
		var t models.Task
		if err := h.db.First(&t, "id = ?", c.Query("taskId")).Error; err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "unknown item"})
			return
		}
		q = q.Where("project_id = ?", t.ProjectID)
		number = t.Number
	case c.Query("repoId") != "":
		q = q.Where("repo_id = ?", c.Query("repoId"))
	case c.Query("projectId") != "":
		q = q.Where("project_id = ?", c.Query("projectId"))
	default:
		c.JSON(http.StatusBadRequest, api.Error{Message: "give a projectId, repoId or taskId"})
		return
	}
	if k := c.Query("kind"); k != "" {
		q = q.Where("kind = ?", k)
	}
	limit := 100
	if number > 0 {
		limit = 3000
	}
	var rows []gitEvent
	q.Order("occurred_at DESC").Limit(limit).Find(&rows)
	names := map[uuid.UUID]string{}
	var repos []projectRepo
	h.db.Select("id, name").Find(&repos)
	for _, r := range repos {
		names[r.ID] = r.Name
	}
	out := []api.GitEvent{}
	for _, e := range rows {
		nums := e.numbers()
		if number > 0 {
			hit := false
			for _, n := range nums {
				hit = hit || n == number
			}
			if !hit {
				continue
			}
		}
		if nums == nil {
			nums = []int{}
		}
		ext := e.ExternalID
		out = append(out, api.GitEvent{Id: e.ID, RepoId: e.RepoID, RepoName: names[e.RepoID], ProjectId: e.ProjectID, Kind: api.GitEventKind(e.Kind),
			ExternalId: &ext, Title: e.Title, Url: e.URL, Author: e.Author, State: e.State, Ref: e.Ref, Sha: e.Sha, Items: nums, OccurredAt: e.OccurredAt})
		if len(out) >= 100 {
			break
		}
	}
	c.JSON(http.StatusOK, api.GitEventList{Data: out})
}

// --- receiving webhooks ---

func parseWhen(s string) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05 MST", "2006-01-02 15:04:05 -0700", "2006-01-02T15:04:05Z07:00"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func sp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func firstLine(s string) string {
	l, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return excerpt(strings.TrimSpace(l), 300)
}

// ReceiveGitHook implements POST /api/v1/git/hooks/{id}.
func (h *Handler) ReceiveGitHook(c *gin.Context) {
	var repo projectRepo
	if err := h.db.First(&repo, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "repository not found"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 5<<20))
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "could not read the body"})
		return
	}
	var events []gitEvent
	if repo.Provider == "github" {
		mac := hmac.New(sha256.New, []byte(repo.Secret))
		mac.Write(body)
		want := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(want), []byte(c.GetHeader("X-Hub-Signature-256"))) {
			c.JSON(http.StatusUnauthorized, api.Error{Message: "bad signature"})
			return
		}
		events, err = githubEvents(c.GetHeader("X-GitHub-Event"), body)
	} else {
		if subtle.ConstantTimeCompare([]byte(repo.Secret), []byte(c.GetHeader("X-Gitlab-Token"))) != 1 {
			c.JSON(http.StatusUnauthorized, api.Error{Message: "bad token"})
			return
		}
		events, err = gitlabEvents(body, repo.URL)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: err.Error()})
		return
	}
	var p models.Project
	h.db.Select("id, key").First(&p, "id = ?", repo.ProjectID)
	for _, e := range events {
		e.RepoID, e.ProjectID = repo.ID, repo.ProjectID
		// Only what people wrote (message, title, description, branch): the
		// titles made here ("#7 …", "Pipeline #12") would look like refs.
		nums := itemNumbers(p.Key, deref(e.Ref), e.Items)
		if e.Kind == "pipeline" && e.Sha != nil {
			var commits []gitEvent
			h.db.Where("repo_id = ? AND kind = ? AND external_id = ?", repo.ID, "commit", *e.Sha).Find(&commits)
			for _, cm := range commits {
				nums = append(nums, cm.numbers()...)
			}
		}
		b, _ := json.Marshal(uniqueInts(nums))
		e.Items = string(b)
		var existing gitEvent
		if h.db.Where("repo_id = ? AND kind = ? AND external_id = ?", repo.ID, e.Kind, e.ExternalID).First(&existing).Error == nil {
			e.ID, e.CreatedAt = existing.ID, existing.CreatedAt
		}
		if err := h.db.Save(&e).Error; err != nil {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to save", Details: strptr(err.Error())})
			return
		}
	}
	now := time.Now()
	h.db.Model(&projectRepo{}).Where("id = ?", repo.ID).Update("last_event_at", now)
	c.JSON(http.StatusAccepted, gin.H{"received": len(events)})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func uniqueInts(in []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, n := range in {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

// Items on a parsed event carries the extra text to scan for item refs
// (full message, description, branch) until it's replaced by the numbers.

func githubEvents(kind string, body []byte) ([]gitEvent, error) {
	switch kind {
	case "ping", "":
		return nil, nil
	case "push":
		var p struct {
			Ref     string `json:"ref"`
			Commits []struct {
				ID        string `json:"id"`
				Message   string `json:"message"`
				URL       string `json:"url"`
				Timestamp string `json:"timestamp"`
				Author    struct {
					Name string `json:"name"`
				} `json:"author"`
			} `json:"commits"`
		}
		if err := json.Unmarshal(body, &p); err != nil {
			return nil, errors.New("not a push payload")
		}
		branch := strings.TrimPrefix(p.Ref, "refs/heads/")
		var out []gitEvent
		for _, cm := range p.Commits {
			out = append(out, gitEvent{Kind: "commit", ExternalID: cm.ID, Title: firstLine(cm.Message), URL: sp(cm.URL), Author: sp(cm.Author.Name),
				Ref: sp(branch), Sha: sp(cm.ID), Items: cm.Message, OccurredAt: parseWhen(cm.Timestamp)})
		}
		return out, nil
	case "pull_request":
		var p struct {
			PR struct {
				Number  int    `json:"number"`
				Title   string `json:"title"`
				Body    string `json:"body"`
				URL     string `json:"html_url"`
				State   string `json:"state"`
				Merged  bool   `json:"merged"`
				Updated string `json:"updated_at"`
				User    struct {
					Login string `json:"login"`
				} `json:"user"`
				Head struct {
					Ref string `json:"ref"`
					Sha string `json:"sha"`
				} `json:"head"`
			} `json:"pull_request"`
		}
		if err := json.Unmarshal(body, &p); err != nil || p.PR.Number == 0 {
			return nil, errors.New("not a pull_request payload")
		}
		state := "open"
		if p.PR.Merged {
			state = "merged"
		} else if p.PR.State == "closed" {
			state = "closed"
		}
		return []gitEvent{{Kind: "pull_request", ExternalID: strconv.Itoa(p.PR.Number), Title: fmt.Sprintf("#%d %s", p.PR.Number, firstLine(p.PR.Title)),
			URL: sp(p.PR.URL), Author: sp(p.PR.User.Login), State: &state, Ref: sp(p.PR.Head.Ref), Sha: sp(p.PR.Head.Sha),
			Items: p.PR.Title + "\n" + p.PR.Body, OccurredAt: parseWhen(p.PR.Updated)}}, nil
	case "workflow_run":
		var p struct {
			Run struct {
				ID         int64  `json:"id"`
				Name       string `json:"name"`
				Status     string `json:"status"`
				Conclusion string `json:"conclusion"`
				URL        string `json:"html_url"`
				Branch     string `json:"head_branch"`
				Sha        string `json:"head_sha"`
				Updated    string `json:"updated_at"`
			} `json:"workflow_run"`
		}
		if err := json.Unmarshal(body, &p); err != nil || p.Run.ID == 0 {
			return nil, errors.New("not a workflow_run payload")
		}
		state := "running"
		if p.Run.Status == "completed" {
			switch p.Run.Conclusion {
			case "success":
				state = "success"
			case "cancelled", "skipped":
				state = "cancelled"
			default:
				state = "failed"
			}
		}
		return []gitEvent{{Kind: "pipeline", ExternalID: strconv.FormatInt(p.Run.ID, 10), Title: p.Run.Name, URL: sp(p.Run.URL), State: &state,
			Ref: sp(p.Run.Branch), Sha: sp(p.Run.Sha), OccurredAt: parseWhen(p.Run.Updated)}}, nil
	}
	return nil, nil // other events are ignored
}

func gitlabEvents(body []byte, repoURL string) ([]gitEvent, error) {
	var head struct {
		Kind string `json:"object_kind"`
	}
	if err := json.Unmarshal(body, &head); err != nil {
		return nil, errors.New("not a GitLab payload")
	}
	switch head.Kind {
	case "push":
		var p struct {
			Ref     string `json:"ref"`
			Commits []struct {
				ID        string `json:"id"`
				Message   string `json:"message"`
				URL       string `json:"url"`
				Timestamp string `json:"timestamp"`
				Author    struct {
					Name string `json:"name"`
				} `json:"author"`
			} `json:"commits"`
		}
		_ = json.Unmarshal(body, &p)
		branch := strings.TrimPrefix(p.Ref, "refs/heads/")
		var out []gitEvent
		for _, cm := range p.Commits {
			out = append(out, gitEvent{Kind: "commit", ExternalID: cm.ID, Title: firstLine(cm.Message), URL: sp(cm.URL), Author: sp(cm.Author.Name),
				Ref: sp(branch), Sha: sp(cm.ID), Items: cm.Message, OccurredAt: parseWhen(cm.Timestamp)})
		}
		return out, nil
	case "merge_request":
		var p struct {
			User struct {
				Name string `json:"name"`
			} `json:"user"`
			MR struct {
				IID         int    `json:"iid"`
				Title       string `json:"title"`
				Description string `json:"description"`
				URL         string `json:"url"`
				State       string `json:"state"`
				Branch      string `json:"source_branch"`
				Updated     string `json:"updated_at"`
				LastCommit  struct {
					ID string `json:"id"`
				} `json:"last_commit"`
			} `json:"object_attributes"`
		}
		_ = json.Unmarshal(body, &p)
		state := map[string]string{"opened": "open", "merged": "merged", "closed": "closed", "locked": "open"}[p.MR.State]
		if state == "" {
			state = "open"
		}
		return []gitEvent{{Kind: "pull_request", ExternalID: strconv.Itoa(p.MR.IID), Title: fmt.Sprintf("!%d %s", p.MR.IID, firstLine(p.MR.Title)),
			URL: sp(p.MR.URL), Author: sp(p.User.Name), State: &state, Ref: sp(p.MR.Branch), Sha: sp(p.MR.LastCommit.ID),
			Items: p.MR.Title + "\n" + p.MR.Description, OccurredAt: parseWhen(p.MR.Updated)}}, nil
	case "pipeline":
		var p struct {
			Attr struct {
				ID       int64  `json:"id"`
				Status   string `json:"status"`
				Ref      string `json:"ref"`
				Sha      string `json:"sha"`
				Finished string `json:"finished_at"`
				Created  string `json:"created_at"`
			} `json:"object_attributes"`
		}
		_ = json.Unmarshal(body, &p)
		state := map[string]string{"success": "success", "failed": "failed", "canceled": "cancelled", "skipped": "cancelled"}[p.Attr.Status]
		if state == "" {
			state = "running"
		}
		when := p.Attr.Finished
		if when == "" {
			when = p.Attr.Created
		}
		u := fmt.Sprintf("%s/-/pipelines/%d", repoURL, p.Attr.ID)
		return []gitEvent{{Kind: "pipeline", ExternalID: strconv.FormatInt(p.Attr.ID, 10), Title: "Pipeline #" + strconv.FormatInt(p.Attr.ID, 10),
			URL: &u, State: &state, Ref: sp(p.Attr.Ref), Sha: sp(p.Attr.Sha), OccurredAt: parseWhen(when)}}, nil
	}
	return nil, nil
}
