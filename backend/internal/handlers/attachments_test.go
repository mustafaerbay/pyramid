package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func upload(t *testing.T, r *gin.Engine, c *http.Cookie, taskID uuid.UUID, name string, content []byte) (int, api.Attachment) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, _ := w.CreateFormFile("file", name)
	fw.Write(content)
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+taskID.String()+"/attachments", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	var a api.Attachment
	if rec.Code == http.StatusCreated {
		json.Unmarshal(rec.Body.Bytes(), &a)
	}
	return rec.Code, a
}

func fetch(r *gin.Engine, c *http.Cookie, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if c != nil {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// A 1×1 PNG.
var tinyPNG = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

func TestAttachments(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "val@x.dev", api.SystemRoleViewer)
	amy := login(t, r, "amy@x.dev", "first-pass")
	bob := login(t, r, "bob@x.dev", "first-pass")
	val := login(t, r, "val@x.dev", "first-pass")
	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", api.TaskCreate{ProjectId: uuid.New(), Title: "T"}, amy, &task)

	s, doc := upload(t, r, amy, task.Id, `../../etc/"notes".html`, []byte("<script>alert(1)</script>"))
	if s != http.StatusCreated {
		t.Fatalf("upload = %d", s)
	}
	if strings.ContainsAny(doc.Filename, `/\"`) || doc.IsImage || doc.SizeBytes != 25 {
		t.Errorf("stored = %+v", doc)
	}
	_, img := upload(t, r, amy, task.Id, "shot.png", tinyPNG)
	if !img.IsImage {
		t.Errorf("png not seen as image: %+v", img)
	}
	if s, _ := upload(t, r, val, task.Id, "x.txt", []byte("x")); s != http.StatusForbidden {
		t.Errorf("viewer upload = %d, want 403", s)
	}
	if s, _ := upload(t, r, amy, uuid.New(), "x.txt", []byte("x")); s != http.StatusNotFound {
		t.Errorf("upload to missing task = %d, want 404", s)
	}
	if s, _ := upload(t, r, amy, task.Id, "big.bin", bytes.Repeat([]byte("a"), 10<<20+1)); s != http.StatusBadRequest {
		t.Errorf("11 MB upload = %d, want 400", s)
	}

	var list api.AttachmentList
	call(t, r, http.MethodGet, "/api/v1/tasks/"+task.Id.String()+"/attachments", nil, val, &list)
	if len(list.Data) != 2 {
		t.Fatalf("list = %d, want 2", len(list.Data))
	}

	// Downloads: signed in only; never rendered as a page.
	if rec := fetch(r, nil, doc.Url); rec.Code != http.StatusUnauthorized {
		t.Errorf("signed-out download = %d", rec.Code)
	}
	rec := fetch(r, val, doc.Url+"?inline=1")
	if rec.Code != http.StatusOK || rec.Body.String() != "<script>alert(1)</script>" {
		t.Fatalf("download = %d %q", rec.Code, rec.Body.String())
	}
	if cd := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(cd, "attachment") {
		t.Errorf("html served as %q, want attachment", cd)
	}
	if rec.Header().Get("Content-Type") != "application/octet-stream" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("html headers = %v", rec.Header())
	}
	rec = fetch(r, val, img.Url+"?inline=1")
	if rec.Header().Get("Content-Type") != "image/png" || !strings.HasPrefix(rec.Header().Get("Content-Disposition"), "inline") {
		t.Errorf("png headers = %v", rec.Header())
	}

	// Deleting: uploader or manager+.
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/attachments/"+doc.Id.String(), nil, bob, nil); s != http.StatusForbidden {
		t.Errorf("bob deletes amy's file = %d, want 403", s)
	}
	if s, _ := call(t, r, http.MethodDelete, "/api/v1/attachments/"+doc.Id.String(), nil, amy, nil); s != http.StatusNoContent {
		t.Errorf("amy deletes = %d", s)
	}
	if rec := fetch(r, val, doc.Url); rec.Code != http.StatusNotFound {
		t.Errorf("download after delete = %d", rec.Code)
	}
}
