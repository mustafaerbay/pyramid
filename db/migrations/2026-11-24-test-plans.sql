-- 2026-11-24: test plans: plans, suites, cases with steps, runs and results.
CREATE TABLE IF NOT EXISTS test_plans (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    sprint_id  UUID REFERENCES sprints(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS test_suites (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id    UUID NOT NULL REFERENCES test_plans(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    position   DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS test_cases (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    suite_id   UUID NOT NULL REFERENCES test_suites(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    steps      JSONB NOT NULL DEFAULT '[]',
    task_id    UUID REFERENCES tasks(id) ON DELETE SET NULL,
    position   DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS test_runs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id      UUID NOT NULL REFERENCES test_plans(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    created_by   UUID REFERENCES people(id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS test_results (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      UUID NOT NULL REFERENCES test_runs(id) ON DELETE CASCADE,
    case_id     UUID REFERENCES test_cases(id) ON DELETE SET NULL,
    title       TEXT NOT NULL,
    steps       JSONB NOT NULL DEFAULT '[]',
    outcome     TEXT NOT NULL DEFAULT 'not_run',
    step_outcomes JSONB NOT NULL DEFAULT '[]',
    comment     TEXT,
    bug_task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    tested_by   UUID REFERENCES people(id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS test_results_run ON test_results(run_id);
