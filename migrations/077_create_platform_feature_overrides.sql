CREATE TABLE platform_feature_overrides (
    feature_key TEXT PRIMARY KEY,
    enabled BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT
);
