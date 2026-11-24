# Pyramid vs Azure DevOps — remaining work

Everything still missing compared to Azure DevOps (ADO), in the order it is
being built. Each item is ticked when it is live (tests pass, pushed, deployed).
Notes say what was built where it is simpler than ADO.

## 1. Boards gaps

- [x] **Teams** — teams inside a project with their own members, area paths and
      sprints; backlog, board and sprints filter by team
- [x] **Per-project permissions** — restricted projects: only their members can
      change them; members can be read-only on a project
- [x] **Process templates** — save a project's process (workflow, types, fields,
      rules, board settings) and apply it to other projects; copy from a project
- [x] **Work item rules** — per state: fields that must be filled; default
      values for new items; read-only fields
- [x] **Backlog levels and forecast** — Epics / Features / Stories / Tasks
      backlog views with rolled-up points, man-days and % done; forecast of the
      sprint each item lands in from recent velocity
- [x] **Sprint taskboard and sprint capacity** — stories as rows, their tasks
      across the states; per-sprint capacity overrides (days off, hours per day,
      activity) per person
- [x] **Board fine settings** — split columns (Doing / Done), definition of done
      per column, which fields cards show, tag colours, card styling rules
- [x] **Query language** — AND/OR groups, custom-field conditions, tree queries
      (items with their children)
- [x] **Charts and dashboards** — pie, stacked bar and trend-over-time charts for
      queries; dashboards with widgets (query chart, burndown, velocity, CFD,
      capacity, notes), personal and shared
- [x] **Delivery plans** — saved plans, dependency lines, markers, team lanes
- [x] **Rich text and @mentions** — Markdown in descriptions and comments,
      person suggestions while typing @
- [x] **Follow items** — follow any item and get notified of its changes
- [x] **Notification subscriptions** — choose which events notify you; Slack /
      Teams incoming webhooks per project
- [x] **Clone, move and templates** — copy an item (with children), move it to
      another project, item templates with pre-filled fields
- [x] **Full-text search** — titles, descriptions and comments, from the header
- [x] **Excel** — .xlsx export and import next to CSV

## 2. Integration and platform

- [x] **API tokens** — personal access tokens for scripts and integrations
- [x] **Webhooks** — outgoing service hooks on item / sprint / comment events
- [x] **Git linking** — GitHub / GitLab webhooks link commits and PRs that mention
      an item (`CAD-12`, `#12` in a project's repo) and show them on the item
- [x] **Two-factor sign-in and SSO** — TOTP 2FA; OpenID Connect sign-in
      (Entra ID, Google, Keycloak …) when configured
- [x] **Audit log** — who changed what across the app (accounts, projects,
      settings), for admins
- [x] **Turkish interface** — Türkçe / English switch (TR/EN in the header; the
      common screens are translated, rarer texts and server messages are still English)

## 3. Other ADO areas

- [x] **Wiki** — Markdown pages per project with a page tree and history
- [x] **Test Plans** — test plans, suites and cases with steps; test runs with
      pass/fail per step; failing step → bug
- [x] **Repos** — repositories linked to a project (GitHub / GitLab), with their
      recent commits and pull requests from the Git webhooks (Pyramid doesn't host Git)
- [x] **Pipelines** — CI/CD run status per project from GitHub Actions / GitLab CI
      webhooks, linked to items and commits (Pyramid doesn't run builds)
- [x] **Artifacts** — with the [arti](../arti) artifact repository: link a project
      to its arti packages, show versions and release notes, tie Pyramid versions
      to arti versions and list the items shipped in each

## 4. Item page

- [x] **Images in descriptions and comments** — paste, drop or pick a screenshot;
      it's attached to the item and shown in the text (click to enlarge); a
      toolbar for bold, italic, code, list and link
- [x] **Item page** — an item opens on its own page (`/projects/…/items/12`):
      description and comments / files / history on the left, fields on the
      right, every change saved as it's made; adding an item stays a quick form
- [x] **Project tabs** — one tab bar on every project page (Overview, Board,
      Backlog, Sprints, Code, Artifacts, Wiki, Tests, Analytics, Settings);
      the long project page split into Overview, Board and Settings; export
      and import in one menu
- [x] **Back keeps the board as it was** — filters last for the session and
      the scroll position comes back after an item
- [x] **UI/UX audit fixes** — no text under 12 px, 24 px minimum targets
      (44 px on touch screens), visible keyboard focus, low-contrast texts
      fixed, SVG icons instead of emoji, split board columns reachable
      from the card's "Move to" menu
- [x] **Main menu in sections** — Home, My Tasks, Projects, Work (Queries,
      Dashboards, Plans), Team (People, Contributions, Growth, Time Off) and
      Admin (Accounts, Audit); a section's pages show as a second row of tabs,
      so the menu fits on a tablet and the page no longer scrolls sideways
- [x] **UI/UX audit of every main page** — secondary text darkened to pass
      4.5:1 everywhere, chart axis text drawn at 12 px at any width, 24 px
      targets in the header and on the people / plans / queries controls,
      labelled search and status filters, no sideways scroll on phones
- [x] **Growth goals and contributions on the person page** — the goals a
      person owns with their progress, and their contributions newest first
      with "Show more"; goals list takes `?ownerPersonId=`
