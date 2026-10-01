-- SPEC-006/B-D: recurring grid, published sessions and audited exceptions.
CREATE TABLE scheduling_series (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'draft',
    target_kind VARCHAR(24) NOT NULL,
    target_id UUID NOT NULL,
    target_name_snapshot VARCHAR(160),
    professional_id UUID NOT NULL,
    professional_name_snapshot VARCHAR(100),
    space_id UUID,
    space_name_snapshot VARCHAR(100),
    delivery_mode VARCHAR(16) NOT NULL,
    capacity INTEGER NOT NULL,
    duration_minutes INTEGER NOT NULL,
    local_start_time TIME NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    effective_from DATE NOT NULL,
    effective_until DATE,
    revision INTEGER NOT NULL DEFAULT 1,
    published_at TIMESTAMPTZ,
    published_by_user_id UUID,
    cancelled_at TIMESTAMPTZ,
    cancelled_by_user_id UUID,
    cancelled_reason VARCHAR(240),
    archived_at TIMESTAMPTZ,
    archived_by_user_id UUID,
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scheduling_series_scope_key UNIQUE (id, organization_id, unit_id),
    CONSTRAINT scheduling_series_unit_fk FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_series_professional_fk FOREIGN KEY (professional_id, organization_id) REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_series_status_check CHECK (status IN ('draft', 'archived', 'published', 'cancelled')),
    CONSTRAINT scheduling_series_target_check CHECK (target_kind IN ('modality', 'service_offering')),
    CONSTRAINT scheduling_series_mode_capacity_check CHECK ((delivery_mode = 'individual' AND capacity = 1) OR (delivery_mode = 'group' AND capacity BETWEEN 1 AND 10000)),
    CONSTRAINT scheduling_series_duration_check CHECK (duration_minutes BETWEEN 5 AND 1440),
    CONSTRAINT scheduling_series_time_check CHECK (date_trunc('minute', local_start_time) = local_start_time),
    CONSTRAINT scheduling_series_timezone_check CHECK (char_length(timezone) BETWEEN 1 AND 64),
    CONSTRAINT scheduling_series_dates_check CHECK (effective_until IS NULL OR effective_until >= effective_from),
    CONSTRAINT scheduling_series_revision_check CHECK (revision > 0),
    CONSTRAINT scheduling_series_archive_check CHECK ((status = 'archived') = (archived_at IS NOT NULL AND archived_by_user_id IS NOT NULL)),
    CONSTRAINT scheduling_series_publish_check CHECK ((status IN ('published', 'cancelled')) = (published_at IS NOT NULL AND published_by_user_id IS NOT NULL)),
    CONSTRAINT scheduling_series_cancel_check CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL AND cancelled_by_user_id IS NOT NULL AND cancelled_reason IS NOT NULL)),
    CONSTRAINT scheduling_series_snapshots_check CHECK ((status IN ('published', 'cancelled')) = (target_name_snapshot IS NOT NULL AND professional_name_snapshot IS NOT NULL))
);

CREATE INDEX idx_scheduling_series_unit_status ON scheduling_series(organization_id, unit_id, status, effective_from, id);
CREATE INDEX idx_scheduling_series_professional ON scheduling_series(organization_id, professional_id, status);
CREATE INDEX idx_scheduling_series_space ON scheduling_series(organization_id, unit_id, space_id, status) WHERE space_id IS NOT NULL;

CREATE TABLE scheduling_series_weekdays (
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    series_id UUID NOT NULL,
    weekday SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    PRIMARY KEY (series_id, weekday),
    CONSTRAINT scheduling_series_weekdays_series_fk FOREIGN KEY (series_id, organization_id, unit_id) REFERENCES scheduling_series(id, organization_id, unit_id) ON DELETE CASCADE
);

CREATE TABLE scheduling_sessions (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    series_id UUID,
    origin VARCHAR(16) NOT NULL,
    occurrence_on DATE,
    status VARCHAR(16) NOT NULL DEFAULT 'scheduled',
    target_kind VARCHAR(24) NOT NULL,
    target_id UUID NOT NULL,
    target_name_snapshot VARCHAR(160) NOT NULL,
    professional_id UUID NOT NULL,
    professional_name_snapshot VARCHAR(100) NOT NULL,
    space_id UUID,
    space_name_snapshot VARCHAR(100),
    delivery_mode VARCHAR(16) NOT NULL,
    capacity INTEGER NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    local_date DATE NOT NULL,
    local_start_time TIME NOT NULL,
    timezone VARCHAR(64) NOT NULL,
    source_series_revision INTEGER,
    manual_override BOOLEAN NOT NULL DEFAULT FALSE,
    revision INTEGER NOT NULL DEFAULT 1,
    published_at TIMESTAMPTZ NOT NULL,
    cancelled_at TIMESTAMPTZ,
    cancelled_by_user_id UUID,
    cancelled_reason VARCHAR(240),
    created_by_user_id UUID NOT NULL,
    updated_by_user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT scheduling_sessions_scope_key UNIQUE (id, organization_id, unit_id),
    CONSTRAINT scheduling_sessions_unit_fk FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_sessions_professional_fk FOREIGN KEY (professional_id, organization_id) REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_sessions_series_fk FOREIGN KEY (series_id, organization_id, unit_id) REFERENCES scheduling_series(id, organization_id, unit_id) ON DELETE RESTRICT,
    CONSTRAINT scheduling_sessions_origin_check CHECK ((origin = 'series' AND series_id IS NOT NULL AND occurrence_on IS NOT NULL) OR (origin = 'extra' AND series_id IS NULL AND occurrence_on IS NULL)),
    CONSTRAINT scheduling_sessions_status_check CHECK (status IN ('scheduled', 'cancelled')),
    CONSTRAINT scheduling_sessions_target_check CHECK (target_kind IN ('modality', 'service_offering')),
    CONSTRAINT scheduling_sessions_mode_capacity_check CHECK ((delivery_mode = 'individual' AND capacity = 1) OR (delivery_mode = 'group' AND capacity BETWEEN 1 AND 10000)),
    CONSTRAINT scheduling_sessions_interval_check CHECK (ends_at > starts_at),
    CONSTRAINT scheduling_sessions_revision_check CHECK (revision > 0),
    CONSTRAINT scheduling_sessions_space_snapshot_check CHECK ((space_id IS NULL) = (space_name_snapshot IS NULL)),
    CONSTRAINT scheduling_sessions_cancel_check CHECK ((status = 'cancelled') = (cancelled_at IS NOT NULL AND cancelled_by_user_id IS NOT NULL AND cancelled_reason IS NOT NULL))
);

CREATE UNIQUE INDEX idx_scheduling_sessions_series_occurrence ON scheduling_sessions(series_id, occurrence_on) WHERE series_id IS NOT NULL;
CREATE INDEX idx_scheduling_sessions_unit_time ON scheduling_sessions(organization_id, unit_id, starts_at, id);
CREATE INDEX idx_scheduling_sessions_professional_time ON scheduling_sessions(organization_id, professional_id, starts_at) WHERE status = 'scheduled';
CREATE INDEX idx_scheduling_sessions_space_time ON scheduling_sessions(organization_id, unit_id, space_id, starts_at) WHERE status = 'scheduled' AND space_id IS NOT NULL;

CREATE TABLE scheduling_session_events (
    sequence BIGSERIAL PRIMARY KEY,
    id UUID NOT NULL UNIQUE,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    session_id UUID NOT NULL,
    session_revision INTEGER NOT NULL,
    event_type VARCHAR(32) NOT NULL CHECK (event_type IN ('session_published', 'session_changed', 'session_cancelled')),
    before_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    after_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    schema_version INTEGER NOT NULL DEFAULT 1 CHECK (schema_version = 1),
    CONSTRAINT scheduling_session_events_session_fk FOREIGN KEY (session_id, organization_id, unit_id) REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT
);
CREATE INDEX idx_scheduling_session_events_scope_sequence ON scheduling_session_events(organization_id, unit_id, sequence);

CREATE TABLE scheduling_materialization_runs (
    id UUID PRIMARY KEY,
    execution_key VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(16) NOT NULL CHECK (status IN ('running', 'completed', 'failed')),
    window_from DATE NOT NULL,
    window_until DATE NOT NULL,
    series_read INTEGER NOT NULL DEFAULT 0,
    sessions_created INTEGER NOT NULL DEFAULT 0,
    sessions_kept INTEGER NOT NULL DEFAULT 0,
    conflicts INTEGER NOT NULL DEFAULT 0,
    error_code VARCHAR(80),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE scheduling_command_requests DROP CONSTRAINT IF EXISTS scheduling_command_requests_professional_organization_fk;
ALTER TABLE scheduling_command_requests DROP CONSTRAINT IF EXISTS scheduling_command_requests_command_check;
ALTER TABLE scheduling_command_requests ALTER COLUMN professional_id DROP NOT NULL;
ALTER TABLE scheduling_command_requests ADD COLUMN series_id UUID;
ALTER TABLE scheduling_command_requests ADD COLUMN session_id UUID;
ALTER TABLE scheduling_command_requests ADD CONSTRAINT scheduling_command_requests_professional_fk FOREIGN KEY (professional_id, organization_id) REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT;
ALTER TABLE scheduling_command_requests ADD CONSTRAINT scheduling_command_requests_series_fk FOREIGN KEY (series_id, organization_id, unit_id) REFERENCES scheduling_series(id, organization_id, unit_id) ON DELETE RESTRICT;
ALTER TABLE scheduling_command_requests ADD CONSTRAINT scheduling_command_requests_session_fk FOREIGN KEY (session_id, organization_id, unit_id) REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT;
ALTER TABLE scheduling_command_requests ADD CONSTRAINT scheduling_command_requests_command_check CHECK (command IN ('professional_create', 'professional_update', 'professional_archive', 'professional_reactivate', 'professional_anonymize', 'series_create', 'series_update', 'series_archive', 'series_reactivate', 'series_publish', 'series_change', 'series_cancel', 'session_create', 'session_change', 'session_cancel'));
ALTER TABLE scheduling_command_requests ADD CONSTRAINT scheduling_command_requests_entity_check CHECK (
    (command LIKE 'professional_%' AND professional_id IS NOT NULL AND series_id IS NULL AND session_id IS NULL) OR
    (command LIKE 'series_%' AND professional_id IS NULL AND series_id IS NOT NULL AND session_id IS NULL) OR
    (command LIKE 'session_%' AND professional_id IS NULL AND series_id IS NULL AND session_id IS NOT NULL)
);

ALTER TABLE scheduling_audit_events DROP CONSTRAINT IF EXISTS scheduling_audit_professional_organization_fk;
ALTER TABLE scheduling_audit_events ADD COLUMN professional_id UUID;
ALTER TABLE scheduling_audit_events ADD COLUMN series_id UUID;
ALTER TABLE scheduling_audit_events ADD COLUMN session_id UUID;
UPDATE scheduling_audit_events SET professional_id = entity_id WHERE entity_type = 'professional';
ALTER TABLE scheduling_audit_events ADD CONSTRAINT scheduling_audit_professional_fk FOREIGN KEY (professional_id, organization_id) REFERENCES scheduling_professionals(id, organization_id) ON DELETE RESTRICT;
ALTER TABLE scheduling_audit_events ADD CONSTRAINT scheduling_audit_series_fk FOREIGN KEY (series_id, organization_id, unit_id) REFERENCES scheduling_series(id, organization_id, unit_id) ON DELETE RESTRICT;
ALTER TABLE scheduling_audit_events ADD CONSTRAINT scheduling_audit_session_fk FOREIGN KEY (session_id, organization_id, unit_id) REFERENCES scheduling_sessions(id, organization_id, unit_id) ON DELETE RESTRICT;
ALTER TABLE scheduling_audit_events ADD CONSTRAINT scheduling_audit_entity_check CHECK (
    (entity_type = 'professional' AND professional_id = entity_id AND series_id IS NULL AND session_id IS NULL) OR
    (entity_type = 'series' AND series_id = entity_id AND professional_id IS NULL AND session_id IS NULL) OR
    (entity_type = 'session' AND session_id = entity_id AND professional_id IS NULL AND series_id IS NULL)
);

CREATE TRIGGER set_scheduling_series_updated_at BEFORE UPDATE ON scheduling_series FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER set_scheduling_sessions_updated_at BEFORE UPDATE ON scheduling_sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
