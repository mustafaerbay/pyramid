# Pyramid vs Azure DevOps — remaining work

Everything still missing compared to Azure DevOps (ADO), in the order it is
being built. Each item is ticked when it is live (tests pass, pushed, deployed).
Notes say what was built where it is simpler than ADO.

## 1. Boards gaps

- [x] **Teams** — teams inside a project with their own members, area paths and
      sprints; backlog, board and sprints filter by team
- [x] **Per-project permissions** — restricted projects: only their members can
      change them; members can be read-only on a project
- [ ] **Process templates** — save a project's process (workflow, types, fields,
      rules, board settings) and apply it to other projects; copy from a project
- [x] **Work item rules** — per state: fields that must be filled; default
      values for new items; read-only fields
- [ ] **Backlog levels and forecast** — Epics / Features / Stories / Tasks
      backlog views with rolled-up points, man-days and % done; forecast of the
      sprint each item lands in from recent velocity
- [ ] **Sprint taskboard and sprint capacity** — stories as rows, their tasks
      across the states; per-sprint capacity overrides (days off, hours per day,
      activity) per person
- [ ] **Board fine settings** — split columns (Doing / Done), definition of done
      per column, which fields cards show, tag colours, card styling rules
- [ ] **Query language** — AND/OR groups, custom-field conditions, tree queries
      (items with their children)
- [ ] **Charts and dashboards** — pie, stacked bar and trend-over-time charts for
      queries; dashboards with widgets (query chart, burndown, velocity, CFD,
      capacity, notes), personal and shared
- [ ] **Delivery plans** — saved plans, dependency lines, markers, team lanes
- [ ] **Rich text and @mentions** — Markdown in descriptions and comments,
      person suggestions while typing @
- [ ] **Follow items** — follow any item and get notified of its changes
- [ ] **Notification subscriptions** — choose which events notify you; Slack /
      Teams incoming webhooks per project
- [ ] **Clone, move and templates** — copy an item (with children), move it to
      another project, item templates with pre-filled fields
- [ ] **Full-text search** — titles, descriptions and comments, from the header
- [ ] **Excel** — .xlsx export and import next to CSV

## 2. Integration and platform

- [ ] **API tokens** — personal access tokens for scripts and integrations
- [ ] **Webhooks** — outgoing service hooks on item / sprint / comment events
- [ ] **Git linking** — GitHub / GitLab webhooks link commits and PRs that mention
      an item (`CAD-12`, `#12` in a project's repo) and show them on the item
- [ ] **Two-factor sign-in and SSO** — TOTP 2FA; OpenID Connect sign-in
      (Entra ID, Google, Keycloak …) when configured
- [ ] **Audit log** — who changed what across the app (accounts, projects,
      settings), for admins
- [ ] **Turkish interface** — Türkçe / English switch

## 3. Other ADO areas

- [ ] **Wiki** — Markdown pages per project with a page tree and history
- [ ] **Test Plans** — test plans, suites and cases with steps; test runs with
      pass/fail per step; failing step → bug
- [ ] **Repos** — repositories linked to a project (GitHub / GitLab), with their
      recent commits and pull requests from the Git webhooks (Pyramid doesn't host Git)
- [ ] **Pipelines** — CI/CD run status per project from GitHub Actions / GitLab CI
      webhooks, linked to items and commits (Pyramid doesn't run builds)
- [ ] **Artifacts** — with the [arti](../arti) artifact repository: link a project
      to its arti packages, show versions and release notes, tie Pyramid versions
      to arti versions and list the items shipped in each
