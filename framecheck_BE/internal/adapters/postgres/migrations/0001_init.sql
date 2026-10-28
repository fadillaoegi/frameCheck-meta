CREATE TABLE projects (
    id         uuid PRIMARY KEY,
    name       text NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
    created_at timestamptz NOT NULL
);

CREATE TABLE assets (
    id                  uuid PRIMARY KEY,
    project_id          uuid NOT NULL REFERENCES projects(id),
    role                text NOT NULL CHECK (role IN ('reference', 'actual')),
    sha256              char(64) NOT NULL,
    mime                text NOT NULL CHECK (mime IN ('image/png', 'image/jpeg')),
    bytes               bigint NOT NULL CHECK (bytes > 0),
    width_px            integer NOT NULL CHECK (width_px > 0),
    height_px           integer NOT NULL CHECK (height_px > 0),
    viewport_width_css  integer NOT NULL CHECK (viewport_width_css > 0),
    viewport_height_css integer NOT NULL CHECK (viewport_height_css > 0),
    device_pixel_ratio  double precision NOT NULL CHECK (device_pixel_ratio > 0),
    content_state_key   text NOT NULL,
    storage_key         text NOT NULL,
    created_at          timestamptz NOT NULL
);
CREATE INDEX assets_project_idx ON assets (project_id, created_at);

CREATE TABLE requirement_versions (
    id         uuid PRIMARY KEY,
    project_id uuid NOT NULL REFERENCES projects(id),
    version    integer NOT NULL CHECK (version > 0),
    text       text NOT NULL,
    sha256     char(64) NOT NULL,
    created_at timestamptz NOT NULL,
    UNIQUE (project_id, version)
);

CREATE TABLE comparisons (
    id                     uuid PRIMARY KEY,
    project_id             uuid NOT NULL REFERENCES projects(id),
    reference_asset_id     uuid NOT NULL REFERENCES assets(id),
    actual_asset_id        uuid NOT NULL REFERENCES assets(id),
    requirement_version_id uuid NOT NULL REFERENCES requirement_versions(id),
    created_at             timestamptz NOT NULL
);
CREATE INDEX comparisons_project_idx ON comparisons (project_id, created_at);

CREATE TABLE runs (
    id                  uuid PRIMARY KEY,
    project_id          uuid NOT NULL REFERENCES projects(id),
    comparison_id       uuid NOT NULL REFERENCES comparisons(id),
    status              text NOT NULL CHECK (status IN ('queued', 'running', 'completed', 'needs_review', 'failed')),
    origin              text NOT NULL CHECK (origin IN ('meta_api', 'fixture', 'mock')),
    input_fingerprint   char(64) NOT NULL,
    idempotency_key     text,
    model_id            text NOT NULL,
    config_version      text NOT NULL,
    prompt_version      text NOT NULL,
    transform           jsonb NOT NULL,
    provider_request_id text NOT NULL DEFAULT '',
    latency_ms          bigint NOT NULL DEFAULT 0,
    failure_code        text NOT NULL DEFAULT '',
    failure_message     text NOT NULL DEFAULT '',
    issues              jsonb NOT NULL DEFAULT '[]',
    created_at          timestamptz NOT NULL,
    started_at          timestamptz,
    finished_at         timestamptz
);
CREATE INDEX runs_comparison_idx ON runs (comparison_id, created_at);
CREATE INDEX runs_fingerprint_idx ON runs (input_fingerprint);
-- One active analysis per project.
CREATE UNIQUE INDEX runs_one_active_per_project ON runs (project_id) WHERE status IN ('queued', 'running');
CREATE UNIQUE INDEX runs_idempotency ON runs (comparison_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE TABLE findings (
    id                 uuid PRIMARY KEY,
    run_id             uuid NOT NULL REFERENCES runs(id),
    position           integer NOT NULL,
    category           text NOT NULL CHECK (category IN ('missing_element', 'clipped_text', 'layout_mismatch')),
    summary            text NOT NULL CHECK (summary <> ''),
    check_step         text NOT NULL DEFAULT '',
    reference_evidence jsonb,
    actual_evidence    jsonb,
    evidence_status    text NOT NULL CHECK (evidence_status IN ('validated', 'needs_review')),
    issues             jsonb NOT NULL DEFAULT '[]',
    created_at         timestamptz NOT NULL,
    UNIQUE (run_id, position)
);

-- Append-only: the latest row per finding is the current decision.
CREATE TABLE reviews (
    id          uuid PRIMARY KEY,
    finding_id  uuid NOT NULL REFERENCES findings(id),
    decision    text NOT NULL CHECK (decision IN ('accepted', 'rejected')),
    reason      text NOT NULL DEFAULT '',
    reviewed_at timestamptz NOT NULL
);
CREATE INDEX reviews_finding_idx ON reviews (finding_id, reviewed_at);
