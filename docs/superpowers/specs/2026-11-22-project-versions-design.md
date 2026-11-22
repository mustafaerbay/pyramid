# Project versions (release plan) — design

Date: 2026-11-22 · Status: draft, awaiting approval

## Intent

Each project gets a list of **versions** (releases such as `v1.0`, `v1.1`).
Milestones are attached to a version. On the project detail page
(`/projects/{id}`) the team can see the **release plan**: what goes into
each version, how far along it is, and whether it will make its date.

Why: the user wants to see the release plan per project. Today there is no
version concept; the only trace is a free-text "Release Version" field in
project info, which holds dummy data and is to be removed.

## Goals

- Define, edit and delete versions inside a project.
- Attach each milestone to at most one version (optional).
- Answer two questions on the project page:
  1. *What is in each version and how much is done?* → **Releases** section.
  2. *Will it make the date?* → the existing milestone timeline, grouped by
     version, with each version's target date drawn as a vertical line.

## Non-goals

- Attaching tasks directly to versions (tasks reach a version through their
  milestone).
- Release notes, changelogs, tagging, deploy integration.
- Versions shared across projects.
- Automatic status changes (status is always set by a person).
- Migrating the old `release_version` text into versions (it is dummy data).

## Anti-goals

- Existing milestones must keep working unchanged when no version is set.
- The release plan must not bury the rest of the page: Releases is one
  compact section, the timeline keeps its current size per milestone.

## Approaches considered

- **A. Separate "Releases" section only** — cards per version with
  milestones and progress. Clear "what's in it", but no sense of dates.
- **B. Timeline only** — group the milestone timeline by version and draw
  release-date lines. Shows date risk, but no room for progress/status, and
  still needs somewhere to create/edit versions.
- **C. Both (chosen by the user)** — Releases cards for content/progress, plus
  timeline grouping and release lines for date risk. B's extra cost is small
  because the timeline already exists.

## Data model

New enum and table in `db/schema.sql` (after `milestones`' dependencies are
satisfied, i.e. the table must be created before `milestones` references it):

```sql
CREATE TYPE version_status AS ENUM ('planned', 'in_progress', 'released', 'cancelled');

CREATE TABLE project_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    status      version_status NOT NULL DEFAULT 'planned',
    target_date DATE,
    released_at DATE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, name)
);
```

`milestones` gets `version_id UUID REFERENCES project_versions(id) ON DELETE SET NULL`
(nullable).

`projects.release_version` is dropped.

- `released_at` follows the milestone `completed_at` pattern: when status
  changes to `released` and no date is given, it is stamped with today. It is
  not cleared automatically if status moves away from `released` (same as
  milestones). *(Builder's call to mirror existing behaviour.)*
- Version name is unique per project → duplicate name returns **409**
  (create and rename). *Builder's call:* check for an existing name inside
  the transaction before insert/update so it works on both Postgres and the
  SQLite tests; the DB `UNIQUE` is the backstop. The existing
  `isUniqueViolation` / 400 behaviour for project members is left alone.

### Released and cancelled versions (decided with the user)

| | Releases section | Timeline | Milestone form select |
|---|---|---|---|
| `planned` / `in_progress` | normal, in sort order | group header + dashed line at `targetDate` | listed |
| `released` | normal, in sort order, shows "Released {releasedAt}" | group header + line at **`releasedAt`** (fallback `targetDate`) | listed |
| `cancelled` | moved to the **end** (before "No version"), muted | group header shown muted, **no line**; its milestones stay visible | listed as "v1.2 (cancelled)" |

- The amber "after release" marker and the red "overdue" date apply only to
  `planned` / `in_progress` versions.
- Sorting: active and released versions by date (`releasedAt` for released,
  else `targetDate`), nulls last, then name; cancelled versions after them;
  "No version" last. The same order drives Releases and the timeline. The
  dates that draw a line also count toward the timeline's month range.

### Migration for the running database

`schema.sql` only runs on an empty Docker volume, and the existing local DB
must keep its data. Add `db/migrations/2026-11-22-project-versions.sql`
containing the enum, table, `ALTER TABLE milestones ADD COLUMN version_id …`,
and `ALTER TABLE projects DROP COLUMN release_version`. Apply it once to the
running DB (`docker exec pyramid-db-1 psql …`). `schema.sql` is updated to
the final shape so fresh volumes get the same result.

## API (`openapi/openapi.yaml`, then `make generate`)

New resource, same shape and conventions as milestones / weekly summaries:

- `GET /api/v1/versions?projectId=…` (paged, `pageSize` up to 100), sorted
  by `target_date ASC NULLS LAST, name ASC`
- `POST /api/v1/versions` — `VersionCreate { projectId, name, description?, targetDate? }`
  (always starts `planned`, like milestones start `pending`)
- `GET / PATCH / DELETE /api/v1/versions/{id}` — `VersionUpdate { name?, description?, status?, targetDate?, releasedAt? }`
- `Version { id, projectId, name, description?, status, targetDate?, releasedAt?, createdAt, updatedAt }`
- New schema enum `VersionStatus`.

Milestones:

- `Milestone`, `MilestoneCreate`, `MilestoneUpdate` gain `versionId` (uuid,
  nullable).
- `PATCH` follows the existing task `milestoneId` pattern: key absent →
  unchanged, explicit `null` → cleared, value → set (`isExplicitNull` in
  `handlers/tasks.go`).
- If `versionId` belongs to a different project than the milestone → **400**.

Deleting a version: the handler sets `version_id = NULL` on its milestones
in the same transaction before deleting (works in the SQLite tests too,
where FK actions aren't relied on); the DB `ON DELETE SET NULL` is the
backstop.

Projects: remove `releaseVersion` from `Project`, `ProjectUpdate` (and any
create schema), the Go model, `ApplyUpdate`/`ToAPI`, and the test schema.

Progress is **not** an API field; the page already loads all tasks and
milestones of the project and computes it (see below).

## Project page

### Releases section (new, directly above Milestones)

```
Releases                                         [+ Add version]
┌──────────────────────────────────────────────────────────────┐
│ v1.0  [in progress]  Target 15 Dec 2026        ███████░░░ 7/12 │
│   ● M1 — Digest email      completed     3/3                  │
│   ● M2 — In-app alerts     in progress   4/6   ⚠ after release │
├──────────────────────────────────────────────────────────────┤
│ v1.1  [planned]      Target 1 Feb 2027         ░░░░░░░░░░ 0/3  │
│   ● M3                     pending       0/3                  │
├──────────────────────────────────────────────────────────────┤
│ No version                                                    │
│   ● M4                     pending       0/0                  │
└──────────────────────────────────────────────────────────────┘
```

- One row block per version, in the order defined above under "Released and
  cancelled versions" (the API's own sort is just a stable default).
- Header: name, status badge (reuse `Badge`), target date; if `released`,
  show "Released {releasedAt}" instead; if target date is past and status is
  `planned`/`in_progress`, show the date in red with "overdue".
- Progress: done tasks / all tasks across the version's milestones — the
  same counting the Tasks section already uses per milestone
  (`status === 'done'` over all tasks). A version with no tasks shows `0/0`
  and an empty bar.
- Under each version: its milestones (status dot, name, milestone status,
  done/total tasks). A milestone whose `dueDate` is after the version's
  `targetDate` gets a small amber "after release" marker.
- Trailing **No version** block lists unassigned milestones; hidden when
  empty.
- Empty state when the project has no versions: "No versions yet. Add one to
  plan releases."
- Clicking a version header opens **VersionFormModal** (edit); clicking a
  milestone opens the existing milestone edit modal.

### VersionFormModal (new component)

Fields: name (required), target date, description; on edit also status and
released date, plus a "Delete version" link → existing `ConfirmDialog`
(text explains its milestones will become "No version"). Mirrors
`MilestoneFormModal` structure and styling.

### Milestone form

`MilestoneFormModal` gets a **Version** select: "— No version —" plus the
project's versions (by name). The page passes the versions list in.

### Milestone timeline (`MilestoneTimeline.svelte`)

- Rows are grouped by version in the same order as the Releases section,
  "No version" last. Each group starts with a thin header row showing the
  version name (and status dot); its milestones follow, one row each, in
  start-date order (the current one-row-per-milestone behaviour).
- Each line-drawing version (see the table above) inside the visible range gets a vertical
  dashed line at its date across the full height, labelled with the
  version name at the top. Those dates also count toward the visible
  month range (like milestone dates do today).
- Undated milestones stay in the existing "No due date set" area.
- If the project has no versions, the timeline looks exactly as today (no
  group headers).

### Project info

Remove the "Release Version" field from the info grid and
`ProjectInfoFormModal`.

## Testing

- Go model tests for `Version` (`ToAPI`, create defaults, `ApplyUpdate`,
  `released_at` stamping) and milestone `versionId` mapping.
- Handler tests (SQLite): version CRUD + list filter/sort, duplicate name →
  409, cross-project `versionId` → 400, `versionId: null` clears, deleting a
  version unassigns its milestones. Add the new table/column to the test
  schema; drop `release_version` there.
- `npm run check` clean; verify the page in the browser with a couple of
  versions (including overdue and "after release" cases).

## Left to the builder

- Exact styling, spacing and copy, following the existing page components.
- File/function names, as long as they follow existing conventions.
- Whether the frontend loads versions with one `pageSize=100` request (fine
  for now; projects won't have >100 versions).
