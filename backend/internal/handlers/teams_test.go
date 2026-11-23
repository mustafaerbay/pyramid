package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/mustafaerbay/pyramid/backend/internal/api"
)

func TestProjectTeams(t *testing.T) {
	r, _ := newTestRouter(t)
	var p api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "T", Name: "Teams"}, &p)
	var web api.Area
	doJSON(t, r, http.MethodPost, "/api/v1/areas", api.AreaCreate{ProjectId: p.Id, Path: "Web"}, &web)
	var otherArea api.Area
	doJSON(t, r, http.MethodPost, "/api/v1/areas", api.AreaCreate{ProjectId: uuid.New(), Path: "X"}, &otherArea)
	ada := uuid.New()

	var team api.ProjectTeam
	if s := doJSON(t, r, http.MethodPost, "/api/v1/project-teams", map[string]any{"projectId": p.Id, "name": " Web team ", "memberIds": []uuid.UUID{ada, ada}, "areaIds": []uuid.UUID{web.Id}}, &team); s != http.StatusCreated {
		t.Fatalf("create = %d", s)
	}
	if team.Name != "Web team" || len(team.MemberIds) != 1 || len(team.AreaIds) != 1 {
		t.Errorf("team = %+v", team)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/project-teams", map[string]any{"projectId": p.Id, "name": "web TEAM"}, nil); s != http.StatusBadRequest {
		t.Errorf("duplicate name = %d", s)
	}
	if s := doJSON(t, r, http.MethodPost, "/api/v1/project-teams", map[string]any{"projectId": p.Id, "name": "B", "areaIds": []uuid.UUID{otherArea.Id}}, nil); s != http.StatusBadRequest {
		t.Errorf("area of another project = %d", s)
	}
	var list api.ProjectTeamList
	doJSON(t, r, http.MethodGet, "/api/v1/project-teams?projectId="+p.Id.String(), nil, &list)
	if len(list.Data) != 1 {
		t.Errorf("list = %d", len(list.Data))
	}
	var upd api.ProjectTeam
	doJSON(t, r, http.MethodPatch, "/api/v1/project-teams/"+team.Id.String(), map[string]any{"name": "Frontend", "memberIds": []uuid.UUID{}, "areaIds": []uuid.UUID{web.Id}}, &upd)
	if upd.Name != "Frontend" || len(upd.MemberIds) != 0 {
		t.Errorf("update = %+v", upd)
	}

	// Sprints can belong to a team of the same project.
	var sp api.Sprint
	if s := doJSON(t, r, http.MethodPost, "/api/v1/sprints", map[string]any{"projectId": p.Id, "teamId": team.Id, "name": "S1", "startDate": "2026-12-01", "endDate": "2026-12-14"}, &sp); s != http.StatusCreated || sp.TeamId == nil || *sp.TeamId != team.Id {
		t.Errorf("sprint with team = %d %+v", s, sp.TeamId)
	}
	var otherTeam api.ProjectTeam
	var p2 api.Project
	doJSON(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "U", Name: "Other"}, &p2)
	doJSON(t, r, http.MethodPost, "/api/v1/project-teams", map[string]any{"projectId": p2.Id, "name": "Them"}, &otherTeam)
	if s := doJSON(t, r, http.MethodPatch, "/api/v1/sprints/"+sp.Id.String(), map[string]any{"teamId": otherTeam.Id}, nil); s != http.StatusBadRequest {
		t.Errorf("team of another project = %d", s)
	}
	doJSON(t, r, http.MethodDelete, "/api/v1/project-teams/"+team.Id.String(), nil, nil)
	var back api.Sprint
	doJSON(t, r, http.MethodGet, "/api/v1/sprints/"+sp.Id.String(), nil, &back)
	if back.TeamId != nil {
		t.Errorf("sprint team after team deleted = %v", back.TeamId)
	}
}

func TestRestrictedProjects(t *testing.T) {
	r, _ := newAuthRouter(t)
	admin := login(t, r, "admin@x.dev", "admin-pass")
	addAccount(t, r, admin, "amy@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "bob@x.dev", api.SystemRoleMember)
	addAccount(t, r, admin, "rea@x.dev", api.SystemRoleMember)
	amy, bob, rea := login(t, r, "amy@x.dev", "first-pass"), login(t, r, "bob@x.dev", "first-pass"), login(t, r, "rea@x.dev", "first-pass")

	var p api.Project
	call(t, r, http.MethodPost, "/api/v1/projects", api.ProjectCreate{Key: "R", Name: "Secret"}, admin, &p)
	call(t, r, http.MethodPost, "/api/v1/project-members", map[string]any{"projectId": p.Id, "personId": personOf(t, r, amy), "role": "contributor"}, admin, nil)
	call(t, r, http.MethodPost, "/api/v1/project-members", map[string]any{"projectId": p.Id, "personId": personOf(t, r, rea), "role": "reviewer", "access": "read"}, admin, nil)
	var task api.Task
	call(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "Before"}, bob, &task)
	if task.Id == uuid.Nil {
		t.Fatal("open project: any member can write")
	}

	if s, _ := call(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"restricted": true}, bob, nil); s != http.StatusForbidden {
		t.Errorf("non-manager restricts = %d, want 403", s)
	}
	call(t, r, http.MethodPatch, "/api/v1/projects/"+p.Id.String(), map[string]any{"restricted": true}, admin, nil)

	tp := "/api/v1/tasks/" + task.Id.String()
	cases := []struct {
		who    string
		cookie *http.Cookie
		want   int
	}{{"member with write", amy, http.StatusOK}, {"outsider", bob, http.StatusForbidden}, {"read-only member", rea, http.StatusForbidden}, {"admin", admin, http.StatusOK}}
	for _, c := range cases {
		if s, _ := call(t, r, http.MethodPatch, tp, map[string]any{"title": "x"}, c.cookie, nil); s != c.want {
			t.Errorf("%s edits = %d, want %d", c.who, s, c.want)
		}
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/tasks", map[string]any{"projectId": p.Id, "title": "New"}, bob, nil); s != http.StatusForbidden {
		t.Errorf("outsider creates = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, tp+"/comments", api.TaskCommentInput{Body: "hi"}, bob, nil); s != http.StatusForbidden {
		t.Errorf("outsider comments = %d", s)
	}
	if s, _ := call(t, r, http.MethodPost, "/api/v1/sprints", map[string]any{"projectId": p.Id, "name": "S", "startDate": "2026-12-01", "endDate": "2026-12-02"}, bob, nil); s != http.StatusForbidden {
		t.Errorf("outsider adds sprint = %d", s)
	}
	if s, _ := call(t, r, http.MethodGet, tp, nil, bob, nil); s != http.StatusOK {
		t.Errorf("outsider reads = %d, want 200", s)
	}
}
