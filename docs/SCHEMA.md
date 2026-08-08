# Pyramid — Data Model

Pyramid is an all-in-one management system for a software department. This document describes the database schema in `db/schema.sql`: what each module holds and how the pieces relate. The schema targets **PostgreSQL** and is written as plain DDL so it drops straight into a Go backend using `golang-migrate`, `sqlc`, or GORM.

## Conventions

Every table uses a `UUID` primary key (`gen_random_uuid()`). Timestamps are `timestamptz`. Mutable tables carry `created_at` and `updated_at`, and `updated_at` is maintained automatically by the `set_updated_at()` trigger. Categorical fields use Postgres `ENUM` types for integrity and compact storage. Foreign-key delete behavior is chosen per relationship — cascades where a child cannot exist without its parent (a task's comments), `SET NULL` where the reference is informational (a project's lead who leaves).

The model is 32 tables across six modules.

## 1. People Management

The backbone the rest of the system references. `departments` form a hierarchy through a self-referencing `parent_id`, and each can name a `lead`. `people` are the humans in the department — every person has a home department and an optional `manager_id` (a self-reference that encodes the reporting line). `people` is deliberately separate from `users`: a person can be tracked (contributions, reviews, org chart) without ever having a login. `users` holds authentication and the `system_role` (admin / manager / member / viewer) that drives access control, and links back one-to-one to a person.

`teams` are squads inside a department; `team_memberships` is the many-to-many join between people and teams, carrying each person's role on the team and their allocation percentage. Capability is tracked through a `skills` catalog and the `person_skills` join, which records a 1–5 proficiency and years of experience per skill.

## 2. Project Management

`projects` are the unit of delivery, each with a short `key` (e.g. `PYR`) used to number tasks, plus status, priority, health, an owning department, an optional `client`, and a lead. `project_members` assigns people to projects with a role and allocation. Work is organized by `milestones` and `sprints` (both belong to a project), and the central `tasks` table hangs off all of them.

A task belongs to a project and carries a per-project sequential `number` (so you get `PYR-42`). Tasks support subtasks and epics through a self-referencing `parent_id`, can be placed in a milestone and a sprint, and record type, status, priority, assignee, reporter, estimates, and story points. `labels` (project-scoped or global) attach through `task_labels`; `task_dependencies` model blocks/depends-on links between tasks; and `task_comments` are threaded via their own `parent_id`.

## 3. Contribution Management

This module answers "who did what." `repositories` link a project to its Git remotes. `commits` and `pull_requests` capture per-repo activity with line counts and authorship tied back to `people`; `pull_request_tasks` connects a PR to the tasks it resolves, and `code_reviews` record who reviewed each PR and the outcome.

Sitting above the specialized tables is `contributions` — a unified, normalized activity feed. Each row is one piece of work (commit, PR, review, design, doc, deployment, …) attributed to a person and optionally to a project, task, or repo, with an `occurred_at`, an optional impact `weight`, and a `jsonb` metadata bag. This is the table dashboards, leaderboards, and per-person rollups query, so you don't have to `UNION` across every source table. `time_entries` records hours logged against tasks and projects, flagged billable or not.

## 4. Growth & Performance

`goals` are OKR-style objectives that can be owned by an individual, team, or department (one of the three owner FKs is set per the `scope`), each broken into measurable `key_results`. `performance_reviews` capture a review cycle for a subject person by a reviewer, with a rating and structured notes. `feedback` is lightweight peer-to-peer kudos or private notes.

## 5. Time Off

`leave_requests` track vacation, sick, parental, and other leave with an approval workflow (status, approver, decision timestamp) and a date-range integrity check.

## 6. System (cross-cutting)

`attachments` is a polymorphic table (`entity_type` + `entity_id`) so any record — a task, a project, a review — can carry files. `notifications` are per-user, typed, and track read state. `activity_log` is a polymorphic audit trail: who did what to which entity, with a `jsonb` field-level diff.

## Key relationships at a glance

- A **person** reports to a manager (self-ref), sits in a department, joins many teams and projects, and holds many skills.
- A **project** owns milestones, sprints, tasks, repositories, and members.
- A **task** belongs to a project, may nest under a parent, and links to comments, labels, dependencies, PRs, contributions, and time entries.
- A **contribution** ties a person to their work and feeds every reporting view.
- **Users** exist only for people who log in; **people** exist for everyone the department tracks.

## Suggested next steps

The natural follow-ups from here are: split this canonical file into ordered `golang-migrate` up/down migrations; generate type-safe Go query code with `sqlc`; add a seed script with representative data; and layer row-level access rules on top of `system_role`. Say the word and I'll scaffold the migrations and the Go project layout next.
