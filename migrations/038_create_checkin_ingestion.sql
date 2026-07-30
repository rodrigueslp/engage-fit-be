CREATE TABLE checkin_ingestion_sources (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(120) NOT NULL,
    source VARCHAR(20) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_ingested_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT checkin_ingestion_sources_source_check CHECK (source IN ('wellhub', 'totalpass')),
    CONSTRAINT checkin_ingestion_sources_box_name_unique UNIQUE (box_id, name)
);

CREATE INDEX idx_checkin_ingestion_sources_box_enabled
    ON checkin_ingestion_sources (box_id, enabled);

CREATE TABLE checkin_ingestion_batches (
    id UUID PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES checkin_ingestion_sources(id) ON DELETE CASCADE,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(128) NOT NULL,
    status VARCHAR(20) NOT NULL,
    import_history_id UUID REFERENCES import_histories(id) ON DELETE SET NULL,
    total_records INTEGER NOT NULL DEFAULT 0,
    students_created INTEGER NOT NULL DEFAULT 0,
    checkins_created INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT checkin_ingestion_batches_status_check CHECK (status IN ('processing', 'completed', 'failed')),
    CONSTRAINT checkin_ingestion_batches_idempotency_unique UNIQUE (source_id, idempotency_key)
);

CREATE INDEX idx_checkin_ingestion_batches_box_created
    ON checkin_ingestion_batches (box_id, created_at DESC);
