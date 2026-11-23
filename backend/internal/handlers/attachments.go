package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"gorm.io/gorm"
)

// Files attached to tasks: rows in attachments, bytes in UPLOAD_DIR named
// by the row id. Downloads need a session and are never rendered as pages:
// everything goes out as a download except real PNG/JPEG/GIF/WebP images
// asked for inline.

const maxUploadBytes = 10 << 20

var previewable = map[string]bool{"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true}

type attachment struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	EntityType string     `gorm:"not null"`
	EntityID   uuid.UUID  `gorm:"type:uuid;not null"`
	Filename   string     `gorm:"not null"`
	URL        string     `gorm:"column:url;not null"`
	MimeType   string     `gorm:"column:mime_type"`
	SizeBytes  int64      `gorm:"column:size_bytes"`
	UploadedBy *uuid.UUID `gorm:"type:uuid;column:uploaded_by"`
	CreatedAt  time.Time
}

func (attachment) TableName() string { return "attachments" }

func (a attachment) toAPI() api.Attachment {
	return api.Attachment{
		Id: a.ID, TaskId: a.EntityID, Filename: a.Filename, MimeType: a.MimeType, SizeBytes: a.SizeBytes,
		UploadedByPersonId: a.UploadedBy, IsImage: previewable[a.MimeType], Url: a.URL, CreatedAt: a.CreatedAt,
	}
}

func (h *Handler) filePath(id uuid.UUID) string { return filepath.Join(h.cfg.UploadDir, id.String()) }

// safeName keeps only the base name and drops characters that could break
// paths or headers.
func safeName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '"' || r == '\\' || unicode.IsControl(r) {
			return '_'
		}
		return r
	}, name)
	name = strings.TrimLeft(strings.TrimSpace(name), ".")
	if utf8.RuneCountInString(name) > 200 {
		name = string([]rune(name)[:200])
	}
	if name == "" {
		name = "file"
	}
	return name
}

// ListTaskAttachments implements GET /api/v1/tasks/{id}/attachments.
func (h *Handler) ListTaskAttachments(c *gin.Context) {
	t, ok := h.loadTask(c)
	if !ok {
		return
	}
	var rows []attachment
	if err := h.db.Where("entity_type = ? AND entity_id = ?", entityTask, t.ID).Order("created_at DESC").Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to list files", Details: strptr(err.Error())})
		return
	}
	out := make([]api.Attachment, len(rows))
	for i, a := range rows {
		out[i] = a.toAPI()
	}
	c.JSON(http.StatusOK, api.AttachmentList{Data: out})
}

// UploadTaskAttachment implements POST /api/v1/tasks/{id}/attachments.
func (h *Handler) UploadTaskAttachment(c *gin.Context) {
	if !h.canWriteTask(c) {
		return
	}
	t, ok := h.loadTask(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "send one file (up to 10 MB) in the \"file\" field"})
		return
	}
	if fh.Size > maxUploadBytes {
		c.JSON(http.StatusBadRequest, api.Error{Message: "files can be at most 10 MB"})
		return
	}
	src, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "could not read the file"})
		return
	}
	defer src.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	mime := http.DetectContentType(head[:n])
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusBadRequest, api.Error{Message: "could not read the file"})
		return
	}

	if err := os.MkdirAll(h.cfg.UploadDir, 0o700); err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "file storage is not available", Details: strptr(err.Error())})
		return
	}
	id := uuid.New()
	dst, err := os.OpenFile(h.filePath(id), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "could not store the file", Details: strptr(err.Error())})
		return
	}
	size, err := io.Copy(dst, src)
	dst.Close()
	if err != nil {
		os.Remove(h.filePath(id))
		c.JSON(http.StatusInternalServerError, api.Error{Message: "could not store the file", Details: strptr(err.Error())})
		return
	}
	a := attachment{
		ID: id, EntityType: entityTask, EntityID: t.ID, Filename: safeName(fh.Filename),
		URL: fmt.Sprintf("/api/v1/attachments/%s/download", id), MimeType: mime, SizeBytes: size,
	}
	if u, ok := currentUser(c); ok {
		pid := u.PersonID
		a.UploadedBy = &pid
	}
	if err := h.db.Create(&a).Error; err != nil {
		os.Remove(h.filePath(id))
		c.JSON(http.StatusInternalServerError, api.Error{Message: "could not save the file", Details: strptr(err.Error())})
		return
	}
	c.JSON(http.StatusCreated, a.toAPI())
}

func (h *Handler) loadAttachment(c *gin.Context) (attachment, bool) {
	var a attachment
	if err := h.db.First(&a, "id = ?", c.Param("id")).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, api.Error{Message: "file not found"})
		} else {
			c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to load file", Details: strptr(err.Error())})
		}
		return a, false
	}
	return a, true
}

// DownloadAttachment implements GET /api/v1/attachments/{id}/download.
func (h *Handler) DownloadAttachment(c *gin.Context) {
	a, ok := h.loadAttachment(c)
	if !ok {
		return
	}
	f, err := os.Open(h.filePath(a.ID))
	if err != nil {
		c.JSON(http.StatusNotFound, api.Error{Message: "file not found"})
		return
	}
	defer f.Close()
	w := c.Writer.Header()
	w.Set("X-Content-Type-Options", "nosniff")
	w.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Set("Cache-Control", "private, max-age=300")
	disposition := "attachment"
	if c.Query("inline") != "" && previewable[a.MimeType] {
		disposition = "inline"
		w.Set("Content-Type", a.MimeType)
	} else {
		w.Set("Content-Type", "application/octet-stream")
	}
	w.Set("Content-Disposition", fmt.Sprintf(`%s; filename="%s"; filename*=UTF-8''%s`,
		disposition, strings.Map(func(r rune) rune {
			if r > 126 {
				return '_'
			}
			return r
		}, a.Filename), url.PathEscape(a.Filename)))
	http.ServeContent(c.Writer, c.Request, "", a.CreatedAt, f)
}

// DeleteAttachment implements DELETE /api/v1/attachments/{id}.
func (h *Handler) DeleteAttachment(c *gin.Context) {
	a, ok := h.loadAttachment(c)
	if !ok {
		return
	}
	mine := false
	if u, ok := currentUser(c); ok && a.UploadedBy != nil && *a.UploadedBy == u.PersonID {
		mine = true
	}
	if !mine && !h.hasRole(c, api.SystemRoleManager) {
		c.JSON(http.StatusForbidden, api.Error{Message: "only the uploader, managers and admins can delete a file"})
		return
	}
	if err := h.db.Delete(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, api.Error{Message: "failed to delete file", Details: strptr(err.Error())})
		return
	}
	os.Remove(h.filePath(a.ID))
	c.Status(http.StatusNoContent)
}

// deleteTaskFiles removes a deleted task's files (best effort).
func (h *Handler) deleteTaskFiles(taskID string) {
	var rows []attachment
	if h.db.Where("entity_type = ? AND entity_id = ?", entityTask, taskID).Find(&rows).Error != nil {
		return
	}
	for _, a := range rows {
		os.Remove(h.filePath(a.ID))
	}
	h.db.Where("entity_type = ? AND entity_id = ?", entityTask, taskID).Delete(&attachment{})
}
