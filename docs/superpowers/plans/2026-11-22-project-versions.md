# Project Versions (Release Plan) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Projects get versions; milestones link to a version; the project page shows a Releases section and a version-grouped milestone timeline.

**Architecture:** New `project_versions` table + `/api/v1/versions` CRUD (same shape as milestones/weekly summaries), `milestones.version_id` FK, `projects.release_version` dropped. Release ordering and progress are pure TS in `frontend/src/lib/releasePlan.ts` (unit-tested), consumed by a new `ReleasePlan.svelte` section and by `MilestoneTimeline.svelte`.

**Tech Stack:** Go 1.25 / Gin / GORM / Postgres 16 (SQLite in handler tests), OpenAPI → oapi-codegen v2.9.0 + openapi-typescript, SvelteKit 2 / Svelte 5 / Tailwind v4, vitest.

**Spec:** `docs/superpowers/specs/2026-11-22-project-versions-design.md`

## Global Constraints

- `openapi/openapi.yaml` is the single source of truth; regenerate with `make generate` (never hand-edit `types.gen.go` / `schema.d.ts`).
- Version status enum values, exactly: `planned`, `in_progress`, `released`, `cancelled`. New versions always start `planned`.
- Version `name` unique per project → **409** on create/rename conflict.
- Milestone `versionId` from another project → **400**.
- PATCH `versionId`: key absent = unchanged, explicit `null` = clear, value = set.
- Deleting a version sets its milestones' `version_id` to NULL (handler, in a transaction; DB `ON DELETE SET NULL` as backstop).
- `released_at` stamped with today when status becomes `released` and none is set/given; never cleared automatically.
- Ordering everywhere (Releases + timeline): non-cancelled by date (`releasedAt` if released, else `targetDate`), nulls last, then name; then cancelled (same sub-order); then "No version".
- "overdue" and "after release" cues only for `planned` / `in_progress`.
- Timeline line date: `releasedAt` (fallback `targetDate`) for released; `targetDate` for planned/in_progress; none for cancelled.
- Progress = tasks with `status === 'done'` / all tasks, across the version's milestones.
- Existing local DB data must survive: changes go through `db/migrations/2026-11-22-project-versions.sql`.
- Commits go straight to `main` (user's standing instruction), with the `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` trailer.

## Review Focus

1. **Milestone with `versionId` pointing at a deleted version, cached in the page** — after deleting a version the page must reload milestones so none still claims the dead id; `groupMilestonesByVersion` must also put a milestone whose `versionId` matches no known version into "No version" rather than dropping it. (Test in Task 4.)
2. **Renaming a version to its own current name** — must not 409. (Test in Task 2.)
3. **Version with no target date and no milestones** — renders in Releases (sorted last among its class) with `0/0`, no timeline line, no crash in month-range math. (Tests in Task 4.)
4. **Milestone created with `versionId` of another project** — 400 on create as well as on update. (Test in Task 3.)
5. **Project with zero versions** — timeline renders exactly as before (no group headers), Releases shows its empty state. (Test in Task 4; checked in browser in Task 6.)

---

### Task 1: Schema, migration, contract, regenerated types; drop `release_version`

**Files:**
- Create: `db/migrations/2026-11-22-project-versions.sql`
- Modify: `db/schema.sql` (enum near the other project enums; `project_versions` table created before `milestones`; `milestones.version_id`; remove `projects.release_version`)
- Modify: `openapi/openapi.yaml` (new `versions` tag, paths, schemas; `versionId` on Milestone*/; remove `releaseVersion` from Project*)
- Regenerate: `backend/internal/api/types.gen.go`, `frontend/src/lib/api/schema.d.ts`
- Modify: `backend/internal/models/project.go` (remove `ReleaseVersion` field, ToAPI, ApplyUpdate lines)
- Modify: `backend/internal/handlers/testutil_test.go` (test schema: drop `release_version`; add `version_id TEXT` to milestones; add `project_versions` table with `UNIQUE (project_id, name)`)
- Modify: `frontend/src/lib/components/ProjectInfoFormModal.svelte`, `frontend/src/routes/projects/[id]/+page.svelte:541` (remove Release Version)

**Interfaces:**
- Produces (Go, generated): `api.Version{Id, ProjectId, Name, Description *string, Status VersionStatus, TargetDate *openapi_types.Date, ReleasedAt *openapi_types.Date, CreatedAt, UpdatedAt}`, `api.VersionCreate{ProjectId, Name, Description *string, TargetDate *Date}`, `api.VersionUpdate{Name *string, Description *string, Status *VersionStatus, TargetDate *Date, ReleasedAt *Date}`, `api.VersionList{Data []Version, Pagination}`, constants `api.VersionStatusPlanned|InProgress|Released|Cancelled`; `VersionId *openapi_types.UUID` on `api.Milestone`, `api.MilestoneCreate`, `api.MilestoneUpdate` (all `nullable: true`).
- Produces (TS): `components['schemas']['Version' | 'VersionStatus' | 'VersionCreate' | 'VersionUpdate']`, `Milestone.versionId?: string | null`; paths `/api/v1/versions`, `/api/v1/versions/{id}` (GET list params `projectId`, `page`, `pageSize`).

- [ ] **Step 1:** Write the migration file: `CREATE TYPE version_status …`, `CREATE TABLE project_versions (…)` exactly as in the spec's Data model, `ALTER TABLE milestones ADD COLUMN version_id UUID REFERENCES project_versions(id) ON DELETE SET NULL`, `ALTER TABLE projects DROP COLUMN release_version`. Mirror the final shape into `db/schema.sql`.
- [ ] **Step 2:** Apply to the running DB and verify:
  Run: `docker exec -i pyramid-db-1 psql -U postgres -d pyramid -v ON_ERROR_STOP=1 < db/migrations/2026-11-22-project-versions.sql && docker exec pyramid-db-1 psql -U postgres -d pyramid -Atc "select count(*) from milestones; select column_name from information_schema.columns where table_name='projects' and column_name='release_version'"`
  Expected: milestone count unchanged (non-zero), no `release_version` row.
- [ ] **Step 3:** Edit `openapi.yaml` (paths mirror `/api/v1/milestones`; list sorted `target_date ASC NULLS LAST, name ASC`; document 409 on POST/PATCH; 400 for cross-project `versionId` on milestones). Run `make generate`.
  Expected: `git diff --stat` shows only the two generated files plus the spec; re-running `make generate` produces no further diff.
- [ ] **Step 4:** Remove `release_version` from the Go model, test schema and the two Svelte files; add test-schema table/column.
- [ ] **Step 5:** Verify: `cd backend && go build ./... && go test ./...` → PASS; `cd frontend && npm run check` → 0 errors.
- [ ] **Step 6:** Commit `feat(versions): schema, migration and API contract; drop release_version` — include the spec and this plan (`docs/superpowers/`) in this commit.

### Task 2: Version model + `/api/v1/versions` handlers

**Files:**
- Create: `backend/internal/models/version.go`, `backend/internal/models/version_test.go`
- Create: `backend/internal/handlers/versions.go`, `backend/internal/handlers/versions_test.go`
- Modify: `backend/internal/handlers/handler.go` (routes, after weekly-summaries)
- Modify: `backend/internal/models/table_name_test.go` (add `Version` → `project_versions`)

**Interfaces:**
- Consumes: Task 1 generated types.
- Produces: `models.Version` (`TableName() = "project_versions"`), `func (v Version) ToAPI() api.Version`, `func NewVersionFromCreate(in api.VersionCreate) Version`, `func (v *Version) ApplyUpdate(in api.VersionUpdate)`; handlers `ListVersions`, `CreateVersion`, `GetVersion`, `UpdateVersion`, `DeleteVersion`; helper `func (h *Handler) versionNameTaken(tx *gorm.DB, projectID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error)`.

- [ ] **Step 1: Write failing model tests** in `version_test.go`:
  - `TestNewVersionFromCreate_AlwaysStartsPlanned` — status == `api.VersionStatusPlanned`, name/description/targetDate copied.
  - `TestVersion_ApplyUpdate_ReleasingStampsReleasedAt` — status→released, `ReleasedAt` == today (UTC date).
  - `TestVersion_ApplyUpdate_ReleasingKeepsExistingReleasedAt` and `..._ExplicitReleasedAtIsRespected`.
  - `TestVersion_ApplyUpdate_LeavingReleasedDoesNotClearReleasedAt`.
  - `TestVersion_ToAPI` — all fields round-trip; nil dates stay nil.
- [ ] **Step 2:** `go test ./internal/models/ -run Version` → FAIL (undefined).
- [ ] **Step 3:** Implement `version.go` mirroring `milestone.go` (completedAt stamping logic → releasedAt).
- [ ] **Step 4:** `go test ./internal/models/` → PASS.
- [ ] **Step 5: Write failing handler tests** in `versions_test.go` (use `newTestRouter`, `doJSON`):
  - `TestVersions_CRUD` — create → 201 status planned; GET → 200; PATCH name/targetDate → 200; DELETE → 204; GET → 404.
  - `TestVersions_List_FilterAndSort` — 2 projects; project A versions `v2 (2027-01-01)`, `v1 (2026-12-01)`, `vX (no date)` → list `?projectId=A` returns names `[v1, v2, vX]`, total 3.
  - `TestVersions_DuplicateNameReturns409` — same name same project → 409 on POST; rename another version to it → 409 on PATCH; same name in another project → 201.
  - `TestVersions_RenameToOwnNameIsOK` — PATCH `{name: <current>}` → 200. *(Review Focus 2)*
  - `TestVersions_DeleteUnassignsMilestones` — insert milestone row with `version_id` via `db.Exec`, DELETE version, `SELECT version_id` → NULL.
- [ ] **Step 6:** `go test ./internal/handlers/ -run Versions` → FAIL.
- [ ] **Step 7:** Implement `versions.go` (copy `weekly_summaries.go` structure). List order: `Order("target_date IS NULL, target_date ASC, name ASC")` (portable NULLS LAST). Create/Update: in `h.db.Transaction`, `versionNameTaken` → 409 `api.Error{Message: "a version with this name already exists in this project"}`. Delete: transaction `UPDATE milestones SET version_id = NULL WHERE version_id = ?` then delete; 404 if no row.
- [ ] **Step 8:** `go test ./...` → PASS; `gofmt -l .` empty.
- [ ] **Step 9:** Commit `feat(versions): version model and CRUD API`.

### Task 3: Milestone ↔ version link

**Files:**
- Modify: `backend/internal/models/milestone.go` (+`VersionID *uuid.UUID gorm:"type:uuid;index"`; ToAPI; NewMilestoneFromCreate; `ApplyUpdate(in api.MilestoneUpdate, clearVersion bool)`)
- Modify: `backend/internal/models/milestone_test.go` (update existing `ApplyUpdate` calls to pass `false`)
- Modify: `backend/internal/handlers/milestones.go`
- Create: `backend/internal/handlers/milestones_test.go`

**Interfaces:**
- Consumes: `models.Version` (Task 2), `isExplicitNull` (`handlers/tasks.go`).
- Produces: `api.Milestone.VersionId` populated; `func (h *Handler) checkVersionInProject(versionID uuid.UUID, projectID uuid.UUID) (ok bool, err error)`.

- [ ] **Step 1: Write failing tests:**
  - model: `TestMilestone_VersionId_CreateUpdateClear` — create with VersionId copies it; `ApplyUpdate({VersionId:&x}, false)` sets; `ApplyUpdate({}, true)` clears; `ApplyUpdate({}, false)` leaves it.
  - handler `milestones_test.go`:
    - `TestMilestones_VersionId_SetAndClear` — create milestone with versionId → response has it; PATCH raw body `{"versionId":null}` → cleared; PATCH `{"name":"x"}` → versionId unchanged.
    - `TestMilestones_VersionFromOtherProject_Returns400` — on POST and on PATCH. *(Review Focus 4)*
    - `TestMilestones_UnknownVersion_Returns400`.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement. `UpdateMilestone` reads the raw body like `UpdateTask` (`c.GetRawData`, unmarshal into `api.MilestoneUpdate` and into `map[string]json.RawMessage`). Validate non-nil `VersionId` with `checkVersionInProject` → 400 `"version does not belong to this project"`.
- [ ] **Step 4:** `go test ./...` → PASS.
- [ ] **Step 5:** Commit `feat(versions): link milestones to versions`.

### Task 4: Release-plan logic (pure TS)

**Files:**
- Create: `frontend/src/lib/releasePlan.ts`, `frontend/src/lib/releasePlan.test.ts`

**Interfaces:**
- Consumes: TS types `Version`, `Milestone`, `Task` from `$lib/api/schema` (only fields: version `id,name,status,targetDate,releasedAt`; milestone `id,versionId,dueDate,startDate,status,name`; task `milestoneId,status`).
- Produces:
  ```ts
  export function sortVersions<V extends VersionLike>(versions: V[]): V[];
  export function versionLineDate(v: VersionLike): string | null;      // per Global Constraints
  export function isVersionActive(v: VersionLike): boolean;             // planned | in_progress
  export function isOverdue(v: VersionLike, today: string): boolean;    // active && targetDate < today
  export function isAfterRelease(m: { dueDate?: string | null }, v: VersionLike): boolean; // active && both dates && m.dueDate > v.targetDate
  export type VersionGroup<V, M> = { version: V | null; milestones: M[] }; // null = "No version"
  export function groupMilestonesByVersion<V extends VersionLike, M extends MilestoneLike>(versions: V[], milestones: M[]): VersionGroup<V, M>[];
  export function taskProgress(milestoneIds: string[], tasks: { milestoneId?: string | null; status: string }[]): { done: number; total: number };
  ```
  `groupMilestonesByVersion` returns groups in `sortVersions` order, every version present even with no milestones, a trailing `version: null` group only if non-empty; milestones inside a group keep input order.

- [ ] **Step 1: Write failing tests** (`describe('releasePlan')`):
  - `sortVersions`: `[v2 planned 2027-01-01, rel released releasedAt 2026-10-01 target 2027-05-01, nodate planned, can cancelled 2026-01-01, v1 in_progress 2026-12-01]` → names `['rel','v1','v2','nodate','can']`.
  - `versionLineDate`: released w/ releasedAt → releasedAt; released w/o → targetDate; cancelled → null; planned → targetDate.
  - `isOverdue('2026-11-22')`: planned target 2026-11-01 → true; released same → false; no target → false.
  - `isAfterRelease`: active v target 2026-12-01, m due 2026-12-02 → true; due equal → false; cancelled v → false.
  - `groupMilestonesByVersion`: milestone with unknown versionId lands in the null group *(Review Focus 1)*; version without milestones still appears *(Review Focus 3)*; zero versions → single null group (or `[]` when no milestones) *(Review Focus 5)*.
  - `taskProgress(['m1','m2'], tasks)` counts only those milestones' tasks; `taskProgress([], tasks)` → `{done:0,total:0}`.
- [ ] **Step 2:** `npm run test -- releasePlan` → FAIL.
- [ ] **Step 3:** Implement.
- [ ] **Step 4:** `npm run test` → PASS.
- [ ] **Step 5:** Commit `feat(versions): release plan ordering and progress helpers`.

### Task 5: Version modal, milestone form select, Releases section

**Files:**
- Create: `frontend/src/lib/components/VersionFormModal.svelte`
- Create: `frontend/src/lib/components/ReleasePlan.svelte`
- Modify: `frontend/src/lib/components/MilestoneFormModal.svelte` (+prop `versions: Version[]`, Version `<select>`)
- Modify: `frontend/src/routes/projects/[id]/+page.svelte` (load versions; Releases section above Milestones; modal + ConfirmDialog wiring; pass versions to milestone modal and timeline)

**Interfaces:**
- Consumes: Task 4 helpers; `Badge`, `ConfirmDialog`, `badgeTones`.
- Produces:
  - `VersionFormModal` props: `{ projectId: string; version?: Version | null; onClose: () => void; onSaved: (v: Version) => void; onDelete?: () => void }` — same structure as `MilestoneFormModal` (status + released date only in edit mode; shows API 409 message as the form error).
  - `ReleasePlan` props: `{ versions: Version[]; milestones: Milestone[]; tasks: Task[]; onSelectVersion: (v: Version) => void; onSelectMilestone: (m: Milestone) => void }`.
  - Page: `let versions = $state<Version[]>([])`, `async function loadVersions()` (`GET /api/v1/versions?projectId=…&pageSize=100`).

- [ ] **Step 1:** `MilestoneFormModal`: select options `— No version —` (value `''`) + versions by name, cancelled labelled `"{name} (cancelled)"`. On edit send `versionId: value || null` (explicit null clears); on create send only when set.
- [ ] **Step 2:** `VersionFormModal` per the spec (status badge tones: planned→`neutral`, in_progress→`warning`, released→`success`, cancelled→`neutralMuted`).
- [ ] **Step 3:** `ReleasePlan` renders `groupMilestonesByVersion` output per the spec's sketch: header (name, status `Badge`, "Target {date}" or "Released {releasedAt}", red "overdue" via `isOverdue`), progress bar + `done/total` from `taskProgress`; milestone rows with status dot, name, status, per-milestone done/total and amber "after release" via `isAfterRelease`; cancelled groups muted (`opacity-60`); empty state `No versions yet. Add one to plan releases.` when `versions.length === 0`.
- [ ] **Step 4:** Page wiring: Releases `<section>` with heading "Releases", subtitle count, `+ Add version` button; delete confirm text `Milestones in "{name}" will move to "No version".`; after version save/delete call `loadVersions()` **and** `loadMilestones()`.
- [ ] **Step 5:** `npm run check` → 0 errors; `npm run test` → PASS.
- [ ] **Step 6:** Commit `feat(versions): releases section, version modal, milestone version select`.

### Task 6: Version-grouped timeline + end-to-end check

**Files:**
- Modify: `frontend/src/lib/components/MilestoneTimeline.svelte`
- Modify: `frontend/src/routes/projects/[id]/+page.svelte` (pass `versions`)

**Interfaces:**
- Consumes: `groupMilestonesByVersion`, `versionLineDate`, `sortVersions` (Task 4).
- Produces: `MilestoneTimeline` new optional prop `versions?: TimelineVersion[]` (`{id,name,status,targetDate?,releasedAt?}`), default `[]`.

- [ ] **Step 1:** Lane assignment: walk `groupMilestonesByVersion(versions, dated)`; when `versions.length > 0`, each group takes one header lane (version name + status dot, muted if cancelled; "No version" for the null group) then one lane per dated milestone in start-date order. With no versions keep today's behaviour exactly (no header lanes). Skip header lanes for groups with no dated milestones *and* no line date.
- [ ] **Step 2:** Release lines: for each version with `versionLineDate` inside the range, a full-height dashed vertical line (`border-l border-dashed`) with the version name label at the top; include those dates in the month-range points.
- [ ] **Step 3:** `npm run check` → 0 errors.
- [ ] **Step 4:** Rebuild and verify in the browser: `docker compose up -d --build backend frontend`; on `/projects/bfb0d64b-2804-4702-8422-414385f19a0b` create `v1.0` (target in ~3 weeks, in progress) and `v1.1` (later), assign M1/M2 to v1.0 and M3 to v1.1, set one milestone due after v1.0 → see "after release"; set a past target → "overdue"; release v1.0 → line moves to released date; cancel v1.1 → muted, no line; delete v1.1 → M3 under "No version". Check a project with no versions looks unchanged. Console: no errors.
- [ ] **Step 5:** Full suite: `cd backend && gofmt -l . && go vet ./... && go test ./...`; `cd frontend && npm run check && npm run test && npm run build` → all pass.
- [ ] **Step 6:** Commit `feat(versions): group timeline by version with release lines`, then `git push origin main` (fetch/rebase first if the remote moved).
