package handlers

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
	"github.com/mustafaerbay/pyramid/backend/internal/models"
)

var itemRef = regexp.MustCompile(`^(?:([A-Za-z][A-Za-z0-9]*)-)?#?(\d{1,7})$`)

func likeEsc(s string) string {
	return "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(s)) + "%"
}

// snippetAround cuts ~160 characters of text around the first word found.
func snippetAround(text string, words []string) string {
	lower := strings.ToLower(text)
	at := -1
	for _, w := range words {
		if i := strings.Index(lower, w); i >= 0 && (at < 0 || i < at) {
			at = i
		}
	}
	r := []rune(oneLine(text))
	if at < 0 {
		at = 0
	}
	// at is a byte index into text; close enough after oneLine for a snippet.
	start := len([]rune(text[:min(at, len(text))])) - 60
	if start < 0 {
		start = 0
	}
	end := min(start+160, len(r))
	if start > len(r) {
		start = 0
	}
	out := string(r[start:end])
	if start > 0 {
		out = "…" + out
	}
	if end < len(r) {
		out += "…"
	}
	return out
}

func hasAll(text string, words []string) bool {
	l := strings.ToLower(text)
	for _, w := range words {
		if !strings.Contains(l, w) {
			return false
		}
	}
	return true
}

// Search implements GET /api/v1/search.
func (h *Handler) Search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" || len(q) > 200 {
		c.JSON(http.StatusBadRequest, api.Error{Message: "type 1–200 characters to search"})
		return
	}
	words := strings.Fields(strings.ToLower(q))
	if len(words) > 6 {
		words = words[:6]
	}
	out := api.SearchResults{Items: []api.SearchItem{}, Projects: []struct {
		Id   uuid.UUID `json:"id"`
		Key  string    `json:"key"`
		Name string    `json:"name"`
	}{}, People: []struct {
		Id   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	}{}}

	var projects []models.Project
	h.db.Select("id, key, name").Find(&projects)
	keyOf := map[uuid.UUID]string{}
	for _, p := range projects {
		keyOf[p.ID] = p.Key
		if hasAll(p.Name+" "+p.Key, words) && len(out.Projects) < 8 {
			out.Projects = append(out.Projects, struct {
				Id   uuid.UUID `json:"id"`
				Key  string    `json:"key"`
				Name string    `json:"name"`
			}{p.ID, p.Key, p.Name})
		}
	}
	var people []models.Person
	pq := h.db.Select("id, first_name, last_name")
	for _, w := range words {
		pq = pq.Where("LOWER(first_name || ' ' || last_name) LIKE ? ESCAPE '\\'", likeEsc(w))
	}
	pq.Limit(8).Find(&people)
	for _, p := range people {
		out.People = append(out.People, struct {
			Id   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		}{p.ID, strings.TrimSpace(p.FirstName + " " + p.LastName)})
	}

	seen := map[uuid.UUID]bool{}
	add := func(t models.Task, in api.SearchItemMatchedIn, snip *string) {
		if seen[t.ID] || len(out.Items) >= 30 {
			return
		}
		seen[t.ID] = true
		out.Items = append(out.Items, api.SearchItem{Id: t.ID, ProjectId: t.ProjectID, ProjectKey: keyOf[t.ProjectID], Number: t.Number,
			Title: t.Title, Type: t.Type, Status: t.Status, MatchedIn: in, Snippet: snip})
	}
	if m := itemRef.FindStringSubmatch(q); m != nil {
		n, _ := strconv.Atoi(m[2])
		var hits []models.Task
		tq := h.db.Where("number = ?", n)
		if m[1] != "" {
			var ids []uuid.UUID
			for id, k := range keyOf {
				if strings.EqualFold(k, m[1]) {
					ids = append(ids, id)
				}
			}
			tq = tq.Where("project_id IN ?", ids)
		}
		tq.Limit(10).Find(&hits)
		for _, t := range hits {
			add(t, api.SearchItemMatchedInNumber, nil)
		}
	}

	tq := h.db.Model(&models.Task{})
	for _, w := range words {
		l := likeEsc(w)
		tq = tq.Where(`(LOWER(title) LIKE ? ESCAPE '\' OR LOWER(COALESCE(description, '')) LIKE ? ESCAPE '\' OR id IN (?))`, l, l,
			h.db.Table("task_comments").Select("task_id").Where(`LOWER(body) LIKE ? ESCAPE '\'`, l))
	}
	var hits []models.Task
	tq.Order("updated_at DESC").Limit(60).Find(&hits)
	rank := func(t models.Task) int {
		switch {
		case hasAll(t.Title, words):
			return 0
		case t.Description != nil && hasAll(t.Title+" "+*t.Description, words):
			return 1
		}
		return 2
	}
	sort.SliceStable(hits, func(i, j int) bool { return rank(hits[i]) < rank(hits[j]) })
	for _, t := range hits {
		switch rank(t) {
		case 0:
			add(t, api.SearchItemMatchedInTitle, nil)
		case 1:
			s := snippetAround(*t.Description, words)
			add(t, api.SearchItemMatchedInDescription, &s)
		default:
			var cm models.TaskComment
			cq := h.db.Where("task_id = ?", t.ID)
			for _, w := range words {
				if !strings.Contains(strings.ToLower(t.Title), w) {
					cq = cq.Where(`LOWER(body) LIKE ? ESCAPE '\'`, likeEsc(w))
				}
			}
			var snip *string
			if cq.Order("created_at DESC").First(&cm).Error == nil {
				s := snippetAround(cm.Body, words)
				snip = &s
			}
			add(t, api.SearchItemMatchedInComment, snip)
		}
	}
	c.JSON(http.StatusOK, out)
}
