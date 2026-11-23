-- 2026-11-24: arti packages linked to projects; which arti version shipped which Pyramid version.
CREATE TABLE IF NOT EXISTS project_packages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    server_url TEXT NOT NULL,
    package    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (project_id, server_url, package)
);
CREATE TABLE IF NOT EXISTS artifact_links (
    package_id       UUID NOT NULL REFERENCES project_packages(id) ON DELETE CASCADE,
    artifact_version TEXT NOT NULL,
    version_id       UUID NOT NULL REFERENCES project_versions(id) ON DELETE CASCADE,
    PRIMARY KEY (package_id, artifact_version)
);
