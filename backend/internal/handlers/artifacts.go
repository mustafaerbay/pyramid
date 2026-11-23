package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

// Artifacts come from arti (an artifact repository for big files). A
// project links its arti packages; Pyramid reads their versions, files and
// release notes from arti's HTTP API and records which arti version
// shipped which Pyramid version (and so which items).

type projectPackage struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ProjectID uuid.UUID `gorm:"type:uuid;not null"`
	ServerURL string    `gorm:"column:server_url;not null"`
	Package   string    `gorm:"not null"`
	CreatedAt time.Time
}

func (projectPackage) TableName() string { return "project_packages" }

type artifactLink struct {
	PackageID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	ArtifactVersion string    `gorm:"primaryKey"`
	VersionID       uuid.UUID `gorm:"type:uuid;not null"`
}

func (artifactLink) TableName() string { return "artifact_links" }

var artiName = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*(/[A-Za-z0-9_][A-Za-z0-9._-]*){0,7}$`)

func pathEsc(pkg string) string {
	parts := strings.Split(pkg, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}

func (p projectPackage) toAPI() api.ProjectPackage {
	return api.ProjectPackage{Id: p.ID, ProjectId: p.ProjectID, ServerUrl: p.ServerURL, Package: p.Package,
		WebUrl: p.ServerURL + "/#/" + pathEsc(p.Package), UpdateUrl: p.ServerURL + "/api/v1/packages/" + pathEsc(p.Package), CreatedAt: p.CreatedAt}
}

// ListProjectPackages implements GET /api/v1/project-packages.
func (h *Handler) ListProjectPackages(c *gin.Context) {
	var rows []projectPackage
	h.db.Where("project_id = ?", c.Query("projectId")).Order("package").Find(&rows)
	out := api.ProjectPackageList{Data: make([]api.ProjectPackage, len(rows))}
	for i, p := range rows {
		out.Data[i] = p.toAPI()
	}
	if h.cfg.ArtiURL != "" {
		d := strings.TrimRight(h.cfg.ArtiURL, "/")
		out.DefaultServerUrl = &d
	}
	c.JSON(http.StatusOK, out)
}

// CreateProjectPackage implements POST /api/v1/project-packages.
func (h *Handler) CreateProjectPackage(c *gin.Context) {
	var in api.ProjectPackageInput
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
	server := h.cfg.ArtiURL
	if in.ServerUrl != nil && strings.TrimSpace(*in.ServerUrl) != "" {
		server = strings.TrimSpace(*in.ServerUrl)
	}
	server = strings.TrimRight(server, "/")
	u, err := url.Parse(server)
	if server == "" || err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(server) > 300 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "give arti's address, e.g. https://arti.example.com"})
		return
	}
	pkg := strings.Trim(strings.TrimSpace(in.Package), "/")
	if !artiName.MatchString(pkg) {
		c.JSON(http.StatusBadRequest, api.Error{Message: "a package is like WMG/Desktop/SmartDock: letters, digits, . _ - and up to 8 folders"})
		return
	}
	p := projectPackage{ProjectID: in.ProjectId, ServerURL: server, Package: pkg}
	if err := h.db.Create(&p).Error; err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "that package is already linked"})
		return
	}
	c.JSON(http.StatusCreated, p.toAPI())
}

func (h *Handler) loadPackage(c *gin.Context) (projectPackage, bool) {
	var p projectPackage
	if err := h.db.First(&p, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "package not found"})
		return p, false
	}
	return p, true
}

// DeleteProjectPackage implements DELETE /api/v1/project-packages/{id}.
func (h *Handler) DeleteProjectPackage(c *gin.Context) {
	p, ok := h.loadPackage(c)
	if !ok {
		return
	}
	h.db.Where("package_id = ?", p.ID).Delete(&artifactLink{})
	h.db.Delete(&p)
	c.Status(http.StatusNoContent)
}

// LinkArtifactVersion implements PUT /api/v1/project-packages/{id}/links.
func (h *Handler) LinkArtifactVersion(c *gin.Context) {
	p, ok := h.loadPackage(c)
	if !ok {
		return
	}
	if !h.canWriteProject(c, p.ProjectID) {
		return
	}
	var in struct {
		ArtifactVersion string     `json:"artifactVersion"`
		VersionID       *uuid.UUID `json:"versionId"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || strings.TrimSpace(in.ArtifactVersion) == "" || len(in.ArtifactVersion) > 100 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "give an artifactVersion"})
		return
	}
	h.db.Where("package_id = ? AND artifact_version = ?", p.ID, in.ArtifactVersion).Delete(&artifactLink{})
	if in.VersionID != nil {
		var v models.Version
		if err := h.db.First(&v, "id = ? AND project_id = ?", *in.VersionID, p.ProjectID).Error; err != nil {
			c.JSON(http.StatusBadRequest, api.Error{Message: "that version isn't in this project"})
			return
		}
		h.db.Create(&artifactLink{PackageID: p.ID, ArtifactVersion: in.ArtifactVersion, VersionID: v.ID})
	}
	c.Status(http.StatusNoContent)
}

var artiClient = &http.Client{Timeout: 8 * time.Second}

func artiGet(ctx context.Context, u string, out any) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	res, err := artiClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, res.StatusCode, err
	}
	if out != nil && res.StatusCode == http.StatusOK {
		err = json.Unmarshal(b, out)
	}
	return b, res.StatusCode, err
}

// ListPackageVersions implements GET /api/v1/project-packages/{id}/versions.
func (h *Handler) ListPackageVersions(c *gin.Context) {
	p, ok := h.loadPackage(c)
	if !ok {
		return
	}
	base := p.ServerURL + "/api/v1"
	var raw []struct {
		Version  string    `json:"version"`
		Created  time.Time `json:"created"`
		HasNotes bool      `json:"has_notes"`
		Locked   bool      `json:"locked"`
		Files    []struct {
			Name   string `json:"name"`
			Size   int64  `json:"size"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	_, status, err := artiGet(c.Request.Context(), base+"/versions/"+pathEsc(p.Package), &raw)
	if status == http.StatusNotFound {
		c.JSON(http.StatusOK, api.ArtifactVersionList{Data: []api.ArtifactVersion{}})
		return
	}
	if err != nil || status != http.StatusOK {
		msg := fmt.Sprintf("arti at %s didn't answer", p.ServerURL)
		if status != 0 {
			msg = fmt.Sprintf("arti answered %d", status)
		}
		c.JSON(http.StatusBadGateway, api.Error{Message: msg})
		return
	}
	var links []artifactLink
	h.db.Where("package_id = ?", p.ID).Find(&links)
	linked := map[string]uuid.UUID{}
	for _, l := range links {
		linked[l.ArtifactVersion] = l.VersionID
	}
	out := api.ArtifactVersionList{Data: make([]api.ArtifactVersion, 0, len(raw))}
	for i, v := range raw {
		av := api.ArtifactVersion{Version: v.Version, Created: v.Created, Locked: v.Locked, Items: []struct {
			Id     uuid.UUID    `json:"id"`
			Number int          `json:"number"`
			Title  string       `json:"title"`
			Type   api.TaskType `json:"type"`
		}{}}
		av.Files = make([]struct {
			Name   string  `json:"name"`
			Sha256 *string `json:"sha256,omitempty"`
			Size   int64   `json:"size"`
			Url    string  `json:"url"`
		}, len(v.Files))
		for j, f := range v.Files {
			sum := f.SHA256
			av.Files[j].Name, av.Files[j].Size, av.Files[j].Sha256 = f.Name, f.Size, &sum
			av.Files[j].Url = base + "/packages/" + pathEsc(p.Package) + "/" + url.PathEscape(v.Version) + "/" + pathEsc(f.Name)
		}
		// Notes for the newest 20 versions (each is one request to arti).
		if v.HasNotes && i < 20 {
			if b, st, err := artiGet(c.Request.Context(), base+"/changelog/"+pathEsc(p.Package)+"/"+url.PathEscape(v.Version), nil); err == nil && st == http.StatusOK {
				av.Notes = string(b)
			}
		}
		if vid, ok := linked[v.Version]; ok {
			var pv models.Version
			if h.db.First(&pv, "id = ?", vid).Error == nil {
				av.VersionId, av.VersionName = &pv.ID, &pv.Name
				var tasks []models.Task
				h.db.Where("milestone_id IN (?) AND status = ?", h.db.Model(&models.Milestone{}).Select("id").Where("version_id = ?", pv.ID), api.TaskStatusDone).
					Order("number").Limit(200).Find(&tasks)
				for _, t := range tasks {
					av.Items = append(av.Items, struct {
						Id     uuid.UUID    `json:"id"`
						Number int          `json:"number"`
						Title  string       `json:"title"`
						Type   api.TaskType `json:"type"`
					}{t.ID, t.Number, t.Title, t.Type})
				}
			}
		}
		out.Data = append(out.Data, av)
	}
	c.JSON(http.StatusOK, out)
}
