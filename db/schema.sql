-- ============================================================================
-- Pyramid — Software Department Management System
-- PostgreSQL schema (canonical DDL)
--
-- Modules:
--   1. People Management        (departments, teams, people, users, skills)
--   2. Project Management        (projects, milestones, sprints, tasks, labels)
--   3. Contribution Management   (repositories, commits, PRs, reviews, time)
--   4. Growth & Performance      (goals/OKRs, reviews, feedback)
--   5. Time Off                  (leave requests)
--   6. System                    (notifications, attachments, activity log)
--
-- Conventions:
--   * UUID primary keys via gen_random_uuid()
--   * timestamptz everywhere; created_at / updated_at on mutable tables
--   * updated_at maintained by trigger (set_updated_at)
--   * ON DELETE rules chosen per relationship (see comments)
--   * All enums are Postgres ENUM types for integrity + small storage
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS pgcrypto;   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS citext;      -- case-insensitive email

-- ---------------------------------------------------------------------------
-- Shared: updated_at trigger
-- ---------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- ENUM TYPES
-- ============================================================================

-- People
CREATE TYPE employment_type AS ENUM ('full_time', 'part_time', 'contractor', 'intern');
CREATE TYPE seniority_level AS ENUM ('intern', 'junior', 'mid', 'senior', 'staff', 'principal', 'lead', 'manager', 'director');
CREATE TYPE person_status   AS ENUM ('active', 'on_leave', 'offboarding', 'inactive');
CREATE TYPE system_role     AS ENUM ('admin', 'manager', 'member', 'viewer');
CREATE TYPE team_role       AS ENUM ('lead', 'member');

-- Projects
CREATE TYPE project_status   AS ENUM ('planning', 'active', 'on_hold', 'completed', 'cancelled');
CREATE TYPE project_priority AS ENUM ('low', 'medium', 'high', 'critical');
CREATE TYPE project_health   AS ENUM ('on_track', 'at_risk', 'off_track');
CREATE TYPE project_role     AS ENUM ('lead', 'maintainer', 'contributor', 'reviewer', 'stakeholder');
CREATE TYPE milestone_status AS ENUM ('pending', 'in_progress', 'completed', 'missed');
CREATE TYPE sprint_status    AS ENUM ('planned', 'active', 'completed', 'cancelled');

-- Tasks
CREATE TYPE task_type     AS ENUM ('epic', 'story', 'task', 'bug', 'subtask', 'chore');
CREATE TYPE task_status   AS ENUM ('backlog', 'todo', 'in_progress', 'in_review', 'blocked', 'done', 'cancelled');
CREATE TYPE task_priority AS ENUM ('low', 'medium', 'high', 'urgent');

-- Contributions
CREATE TYPE contribution_type AS ENUM ('commit', 'pull_request', 'code_review', 'issue', 'design', 'documentation', 'deployment', 'support', 'other');
CREATE TYPE pr_status         AS ENUM ('draft', 'open', 'merged', 'closed');
CREATE TYPE review_state      AS ENUM ('approved', 'changes_requested', 'commented', 'dismissed', 'pending');
CREATE TYPE repo_provider     AS ENUM ('github', 'gitlab', 'bitbucket', 'gitea', 'other');

-- Growth & performance
CREATE TYPE goal_scope   AS ENUM ('individual', 'team', 'department', 'company');
CREATE TYPE goal_status  AS ENUM ('draft', 'active', 'achieved', 'partially_achieved', 'missed', 'cancelled');
CREATE TYPE review_cycle AS ENUM ('monthly', 'quarterly', 'biannual', 'annual', 'probation', 'ad_hoc');

-- Time off
CREATE TYPE leave_type   AS ENUM ('vacation', 'sick', 'personal', 'parental', 'bereavement', 'unpaid', 'other');
CREATE TYPE leave_status AS ENUM ('pending', 'approved', 'rejected', 'cancelled');

-- System
CREATE TYPE notification_type AS ENUM ('mention', 'assignment', 'status_change', 'review_request', 'approval', 'deadline', 'system');

-- ============================================================================
-- 1. PEOPLE MANAGEMENT
-- ============================================================================

-- Organizational units, self-referencing for hierarchy (dept -> sub-dept)
CREATE TABLE departments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    description TEXT,
    parent_id   UUID REFERENCES departments(id) ON DELETE SET NULL,
    lead_id     UUID,                -- FK added after people table exists
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- People = the humans in the department (may or may not have a login account)
CREATE TABLE people (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name        TEXT NOT NULL,
    last_name         TEXT NOT NULL,
    email             CITEXT NOT NULL UNIQUE,
    phone             TEXT,
    avatar_url        TEXT,
    title             TEXT,                                       -- free-text job title
    seniority         seniority_level,
    employment_type   employment_type NOT NULL DEFAULT 'full_time',
    status            person_status   NOT NULL DEFAULT 'active',
    department_id     UUID REFERENCES departments(id) ON DELETE SET NULL,
    manager_id        UUID REFERENCES people(id)      ON DELETE SET NULL,  -- reporting line
    location          TEXT,
    timezone          TEXT,
    bio               TEXT,
    hire_date         DATE,
    termination_date  DATE,
    weekly_capacity_hours NUMERIC(5,2) DEFAULT 40,               -- for allocation math
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- department lead FK (defined here to break circular dependency)
ALTER TABLE departments
    ADD CONSTRAINT departments_lead_fk
    FOREIGN KEY (lead_id) REFERENCES people(id) ON DELETE SET NULL;

-- Login accounts / auth. Separated from people so non-users can still be tracked.
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id      UUID NOT NULL UNIQUE REFERENCES people(id) ON DELETE CASCADE,
    email          CITEXT NOT NULL UNIQUE,
    password_hash  TEXT,                          -- null if SSO-only
    system_role    system_role NOT NULL DEFAULT 'member',
    is_active      BOOLEAN NOT NULL DEFAULT true,
    last_login_at  TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Teams / squads within a department
CREATE TABLE teams (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL UNIQUE,
    description   TEXT,
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    lead_id       UUID REFERENCES people(id)      ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Many-to-many: people belong to teams (with a role and allocation window)
CREATE TABLE team_memberships (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id        UUID NOT NULL REFERENCES teams(id)   ON DELETE CASCADE,
    person_id      UUID NOT NULL REFERENCES people(id)  ON DELETE CASCADE,
    role           team_role NOT NULL DEFAULT 'member',
    allocation_pct NUMERIC(5,2) NOT NULL DEFAULT 100 CHECK (allocation_pct >= 0 AND allocation_pct <= 100),
    joined_at      DATE NOT NULL DEFAULT CURRENT_DATE,
    left_at        DATE,
    UNIQUE (team_id, person_id)
);

-- Skills catalog
CREATE TABLE skills (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    category   TEXT,                     -- e.g. 'language', 'framework', 'soft', 'ops'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Person <-> skill with proficiency
CREATE TABLE person_skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    skill_id    UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    proficiency SMALLINT NOT NULL DEFAULT 3 CHECK (proficiency BETWEEN 1 AND 5),
    years_experience NUMERIC(4,1),
    is_primary  BOOLEAN NOT NULL DEFAULT false,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (person_id, skill_id)
);

-- ============================================================================
-- 2. PROJECT MANAGEMENT
-- ============================================================================

-- Clients / customers a project is delivered for (internal or external)
CREATE TABLE clients (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    is_internal  BOOLEAN NOT NULL DEFAULT true,
    contact_name TEXT,
    contact_email CITEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key           TEXT NOT NULL UNIQUE,           -- short code, e.g. 'PYR' used for task numbers
    name          TEXT NOT NULL,
    description   TEXT,
    status        project_status   NOT NULL DEFAULT 'planning',
    priority      project_priority NOT NULL DEFAULT 'medium',
    health        project_health   NOT NULL DEFAULT 'on_track',
    department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    client_id     UUID REFERENCES clients(id)     ON DELETE SET NULL,
    lead_id       UUID REFERENCES people(id)      ON DELETE SET NULL,
    start_date    DATE,
    target_date   DATE,
    completed_at  DATE,
    budget_amount NUMERIC(14,2),
    budget_currency CHAR(3) DEFAULT 'USD',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Who is on the project and at what allocation
CREATE TABLE project_members (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id     UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    person_id      UUID NOT NULL REFERENCES people(id)   ON DELETE CASCADE,
    role           project_role NOT NULL DEFAULT 'contributor',
    allocation_pct NUMERIC(5,2) NOT NULL DEFAULT 100 CHECK (allocation_pct >= 0 AND allocation_pct <= 100),
    joined_at      DATE NOT NULL DEFAULT CURRENT_DATE,
    left_at        DATE,
    UNIQUE (project_id, person_id)
);

CREATE TABLE milestones (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT,
    status       milestone_status NOT NULL DEFAULT 'pending',
    start_date   DATE,
    due_date     DATE,
    completed_at DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sprints (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    goal       TEXT,
    status     sprint_status NOT NULL DEFAULT 'planned',
    start_date DATE,
    end_date   DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id    UUID NOT NULL REFERENCES projects(id)   ON DELETE CASCADE,
    number        INTEGER NOT NULL,                        -- per-project sequential (PYR-42)
    parent_id     UUID REFERENCES tasks(id)      ON DELETE SET NULL,  -- subtasks / epic children
    milestone_id  UUID REFERENCES milestones(id) ON DELETE SET NULL,
    sprint_id     UUID REFERENCES sprints(id)    ON DELETE SET NULL,
    title         TEXT NOT NULL,
    description   TEXT,
    type          task_type     NOT NULL DEFAULT 'task',
    status        task_status   NOT NULL DEFAULT 'backlog',
    priority      task_priority NOT NULL DEFAULT 'medium',
    assignee_id   UUID REFERENCES people(id) ON DELETE SET NULL,
    reporter_id   UUID REFERENCES people(id) ON DELETE SET NULL,
    story_points  NUMERIC(5,2),
    estimate_hours NUMERIC(6,2),
    due_date      DATE,
    started_at    TIMESTAMPTZ,
    completed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, number)
);

CREATE TABLE labels (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    color      TEXT NOT NULL DEFAULT '#888888',
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,  -- null = global label
    UNIQUE (project_id, name)
);

CREATE TABLE task_labels (
    task_id  UUID NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    label_id UUID NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, label_id)
);

-- Task blocks/depends-on relationships
CREATE TABLE task_dependencies (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,  -- the blocked task
    depends_on_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,  -- must finish first
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (task_id, depends_on_id),
    CHECK (task_id <> depends_on_id)
);

-- Threaded comments on tasks
CREATE TABLE task_comments (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id    UUID NOT NULL REFERENCES tasks(id)  ON DELETE CASCADE,
    author_id  UUID REFERENCES people(id)          ON DELETE SET NULL,
    parent_id  UUID REFERENCES task_comments(id)   ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- 3. CONTRIBUTION MANAGEMENT
-- ============================================================================

CREATE TABLE repositories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID REFERENCES projects(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    provider    repo_provider NOT NULL DEFAULT 'github',
    url         TEXT NOT NULL,
    default_branch TEXT NOT NULL DEFAULT 'main',
    is_archived BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, url)
);

CREATE TABLE commits (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    sha           TEXT NOT NULL,
    author_id     UUID REFERENCES people(id) ON DELETE SET NULL,
    message       TEXT,
    additions     INTEGER NOT NULL DEFAULT 0,
    deletions     INTEGER NOT NULL DEFAULT 0,
    files_changed INTEGER NOT NULL DEFAULT 0,
    committed_at  TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (repository_id, sha)
);

CREATE TABLE pull_requests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    repository_id UUID NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
    number        INTEGER NOT NULL,
    title         TEXT NOT NULL,
    author_id     UUID REFERENCES people(id) ON DELETE SET NULL,
    status        pr_status NOT NULL DEFAULT 'open',
    additions     INTEGER NOT NULL DEFAULT 0,
    deletions     INTEGER NOT NULL DEFAULT 0,
    commits_count INTEGER NOT NULL DEFAULT 0,
    url           TEXT,
    opened_at     TIMESTAMPTZ,
    merged_at     TIMESTAMPTZ,
    closed_at     TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (repository_id, number)
);

-- Link PRs to the tasks they resolve (many-to-many)
CREATE TABLE pull_request_tasks (
    pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    task_id         UUID NOT NULL REFERENCES tasks(id)         ON DELETE CASCADE,
    PRIMARY KEY (pull_request_id, task_id)
);

CREATE TABLE code_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pull_request_id UUID NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    reviewer_id     UUID REFERENCES people(id) ON DELETE SET NULL,
    state           review_state NOT NULL DEFAULT 'pending',
    body            TEXT,
    submitted_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unified contribution log — a normalized activity feed across all work types.
-- Specialized tables above hold rich data; this gives a single queryable stream
-- for dashboards, leaderboards, and per-person contribution rollups.
CREATE TABLE contributions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id     UUID NOT NULL REFERENCES people(id)     ON DELETE CASCADE,
    project_id    UUID REFERENCES projects(id)            ON DELETE SET NULL,
    task_id       UUID REFERENCES tasks(id)               ON DELETE SET NULL,
    repository_id UUID REFERENCES repositories(id)        ON DELETE SET NULL,
    type          contribution_type NOT NULL,
    title         TEXT,
    url           TEXT,
    weight        NUMERIC(6,2) NOT NULL DEFAULT 1,   -- optional impact/weighting
    occurred_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Logged time against tasks / projects
CREATE TABLE time_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   UUID NOT NULL REFERENCES people(id)   ON DELETE CASCADE,
    project_id  UUID REFERENCES projects(id)          ON DELETE SET NULL,
    task_id     UUID REFERENCES tasks(id)             ON DELETE SET NULL,
    entry_date  DATE NOT NULL DEFAULT CURRENT_DATE,
    hours       NUMERIC(5,2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    description TEXT,
    is_billable BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- 4. GROWTH & PERFORMANCE
-- ============================================================================

-- Objectives (OKR-style). Owner is a person OR a team OR a department.
CREATE TABLE goals (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT NOT NULL,
    description   TEXT,
    scope         goal_scope  NOT NULL DEFAULT 'individual',
    status        goal_status NOT NULL DEFAULT 'draft',
    owner_person_id UUID REFERENCES people(id)      ON DELETE SET NULL,
    owner_team_id   UUID REFERENCES teams(id)       ON DELETE SET NULL,
    owner_department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    progress_pct  NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (progress_pct BETWEEN 0 AND 100),
    start_date    DATE,
    due_date      DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE key_results (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    goal_id       UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
    description   TEXT NOT NULL,
    target_value  NUMERIC(14,2),
    current_value NUMERIC(14,2) NOT NULL DEFAULT 0,
    unit          TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Performance reviews
CREATE TABLE performance_reviews (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id   UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,   -- who is reviewed
    reviewer_id  UUID REFERENCES people(id) ON DELETE SET NULL,           -- who reviews
    cycle        review_cycle NOT NULL DEFAULT 'quarterly',
    period_start DATE,
    period_end   DATE,
    rating       SMALLINT CHECK (rating BETWEEN 1 AND 5),
    summary      TEXT,
    strengths    TEXT,
    improvements TEXT,
    is_finalized BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lightweight peer feedback / kudos
CREATE TABLE feedback (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_id     UUID REFERENCES people(id) ON DELETE SET NULL,
    to_id       UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    message     TEXT NOT NULL,
    is_private  BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- 5. TIME OFF
-- ============================================================================

CREATE TABLE leave_requests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id    UUID NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    type         leave_type   NOT NULL DEFAULT 'vacation',
    status       leave_status NOT NULL DEFAULT 'pending',
    start_date   DATE NOT NULL,
    end_date     DATE NOT NULL,
    reason       TEXT,
    approver_id  UUID REFERENCES people(id) ON DELETE SET NULL,
    decided_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

-- ============================================================================
-- 6. SYSTEM (cross-cutting)
-- ============================================================================

-- Polymorphic attachments (entity_type + entity_id point at any row)
CREATE TABLE attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type  TEXT NOT NULL,           -- e.g. 'task', 'project', 'performance_review'
    entity_id    UUID NOT NULL,
    filename     TEXT NOT NULL,
    url          TEXT NOT NULL,
    mime_type    TEXT,
    size_bytes   BIGINT,
    uploaded_by  UUID REFERENCES people(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE notifications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       notification_type NOT NULL,
    title      TEXT NOT NULL,
    body       TEXT,
    link_url   TEXT,
    read_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Audit / activity trail (polymorphic target + JSON diff)
CREATE TABLE activity_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    action      TEXT NOT NULL,            -- 'created' | 'updated' | 'deleted' | custom
    entity_type TEXT NOT NULL,
    entity_id   UUID,
    changes     JSONB,                    -- { field: { from, to } }
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================================
-- INDEXES (foreign keys + common query paths)
-- ============================================================================

-- People
CREATE INDEX idx_departments_parent      ON departments(parent_id);
CREATE INDEX idx_people_department       ON people(department_id);
CREATE INDEX idx_people_manager          ON people(manager_id);
CREATE INDEX idx_people_status           ON people(status);
CREATE INDEX idx_teams_department        ON teams(department_id);
CREATE INDEX idx_team_memberships_person ON team_memberships(person_id);
CREATE INDEX idx_team_memberships_team   ON team_memberships(team_id);
CREATE INDEX idx_person_skills_person    ON person_skills(person_id);
CREATE INDEX idx_person_skills_skill     ON person_skills(skill_id);

-- Projects
CREATE INDEX idx_projects_status         ON projects(status);
CREATE INDEX idx_projects_lead           ON projects(lead_id);
CREATE INDEX idx_projects_department     ON projects(department_id);
CREATE INDEX idx_project_members_person  ON project_members(person_id);
CREATE INDEX idx_project_members_project ON project_members(project_id);
CREATE INDEX idx_milestones_project      ON milestones(project_id);
CREATE INDEX idx_sprints_project         ON sprints(project_id);
CREATE INDEX idx_tasks_project           ON tasks(project_id);
CREATE INDEX idx_tasks_assignee          ON tasks(assignee_id);
CREATE INDEX idx_tasks_status            ON tasks(status);
CREATE INDEX idx_tasks_sprint            ON tasks(sprint_id);
CREATE INDEX idx_tasks_milestone         ON tasks(milestone_id);
CREATE INDEX idx_tasks_parent            ON tasks(parent_id);
CREATE INDEX idx_task_comments_task      ON task_comments(task_id);

-- Contributions
CREATE INDEX idx_repositories_project    ON repositories(project_id);
CREATE INDEX idx_commits_repo            ON commits(repository_id);
CREATE INDEX idx_commits_author          ON commits(author_id);
CREATE INDEX idx_commits_committed_at    ON commits(committed_at);
CREATE INDEX idx_prs_repo                ON pull_requests(repository_id);
CREATE INDEX idx_prs_author              ON pull_requests(author_id);
CREATE INDEX idx_prs_status              ON pull_requests(status);
CREATE INDEX idx_code_reviews_pr         ON code_reviews(pull_request_id);
CREATE INDEX idx_code_reviews_reviewer   ON code_reviews(reviewer_id);
CREATE INDEX idx_contributions_person    ON contributions(person_id);
CREATE INDEX idx_contributions_project   ON contributions(project_id);
CREATE INDEX idx_contributions_type      ON contributions(type);
CREATE INDEX idx_contributions_occurred  ON contributions(occurred_at);
CREATE INDEX idx_time_entries_person     ON time_entries(person_id);
CREATE INDEX idx_time_entries_project    ON time_entries(project_id);
CREATE INDEX idx_time_entries_date       ON time_entries(entry_date);

-- Growth
CREATE INDEX idx_goals_owner_person      ON goals(owner_person_id);
CREATE INDEX idx_goals_owner_team        ON goals(owner_team_id);
CREATE INDEX idx_key_results_goal        ON key_results(goal_id);
CREATE INDEX idx_perf_reviews_subject    ON performance_reviews(subject_id);
CREATE INDEX idx_feedback_to             ON feedback(to_id);

-- Time off
CREATE INDEX idx_leave_person            ON leave_requests(person_id);
CREATE INDEX idx_leave_status            ON leave_requests(status);

-- System
CREATE INDEX idx_attachments_entity      ON attachments(entity_type, entity_id);
CREATE INDEX idx_notifications_user      ON notifications(user_id, read_at);
CREATE INDEX idx_activity_entity         ON activity_log(entity_type, entity_id);
CREATE INDEX idx_activity_actor          ON activity_log(actor_id);

-- ============================================================================
-- updated_at TRIGGERS (mutable tables)
-- ============================================================================
CREATE TRIGGER trg_departments_updated  BEFORE UPDATE ON departments         FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_people_updated        BEFORE UPDATE ON people              FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_users_updated         BEFORE UPDATE ON users               FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_teams_updated         BEFORE UPDATE ON teams               FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_person_skills_updated BEFORE UPDATE ON person_skills       FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_clients_updated       BEFORE UPDATE ON clients             FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_projects_updated      BEFORE UPDATE ON projects            FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_milestones_updated    BEFORE UPDATE ON milestones          FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_sprints_updated       BEFORE UPDATE ON sprints             FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_tasks_updated         BEFORE UPDATE ON tasks               FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_task_comments_updated BEFORE UPDATE ON task_comments       FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_repositories_updated  BEFORE UPDATE ON repositories        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_prs_updated           BEFORE UPDATE ON pull_requests       FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_time_entries_updated  BEFORE UPDATE ON time_entries        FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_goals_updated         BEFORE UPDATE ON goals               FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_key_results_updated   BEFORE UPDATE ON key_results         FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_perf_reviews_updated  BEFORE UPDATE ON performance_reviews FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_leave_updated         BEFORE UPDATE ON leave_requests      FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================
