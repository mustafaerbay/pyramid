-- 2026-11-24: project wiki: Markdown pages in a tree, with history.
CREATE TABLE IF NOT EXISTS wiki_pages (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    parent_id  UUID REFERENCES wiki_pages(id) ON DELETE SET NULL,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL DEFAULT '',
    position   DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_by UUID REFERENCES people(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS wiki_pages_project ON wiki_pages(project_id);
CREATE TABLE IF NOT EXISTS wiki_revisions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    page_id    UUID NOT NULL REFERENCES wiki_pages(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    body       TEXT NOT NULL,
    author_id  UUID REFERENCES people(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS wiki_revisions_page ON wiki_revisions(page_id, created_at DESC);
