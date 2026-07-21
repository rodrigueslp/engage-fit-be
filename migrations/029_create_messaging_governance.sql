ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ALTER COLUMN box_id DROP NOT NULL;
ALTER TABLE users
    ADD CONSTRAINT users_role_check CHECK (
        (role = 'OWNER' AND box_id IS NOT NULL) OR
        (role = 'PLATFORM_ADMIN' AND box_id IS NULL)
    );

CREATE TABLE messaging_policies (
    id UUID PRIMARY KEY,
    scope VARCHAR(20) NOT NULL,
    box_id UUID REFERENCES boxes(id) ON DELETE CASCADE,
    daily_message_limit INTEGER NOT NULL,
    monthly_message_limit INTEGER NOT NULL,
    per_dispatch_limit INTEGER NOT NULL,
    estimated_cost_micros_per_message BIGINT NOT NULL,
    daily_cost_limit_micros BIGINT NOT NULL,
    monthly_cost_limit_micros BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    warning_percent INTEGER NOT NULL DEFAULT 80,
    timezone VARCHAR(64) NOT NULL DEFAULT 'America/Sao_Paulo',
    blocked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT messaging_policies_scope_check CHECK (scope IN ('box', 'platform')),
    CONSTRAINT messaging_policies_owner_check CHECK (
        (scope = 'box' AND box_id IS NOT NULL) OR
        (scope = 'platform' AND box_id IS NULL)
    ),
    CONSTRAINT messaging_policies_limits_check CHECK (
        daily_message_limit >= 0 AND monthly_message_limit >= 0 AND
        per_dispatch_limit >= 0 AND estimated_cost_micros_per_message >= 0 AND
        daily_cost_limit_micros >= 0 AND monthly_cost_limit_micros >= 0 AND
        warning_percent BETWEEN 1 AND 100
    )
);

CREATE UNIQUE INDEX idx_messaging_policies_box ON messaging_policies(box_id) WHERE scope = 'box';
CREATE UNIQUE INDEX idx_messaging_policies_platform ON messaging_policies(scope) WHERE scope = 'platform';

INSERT INTO messaging_policies (
    id, scope, box_id, daily_message_limit, monthly_message_limit,
    per_dispatch_limit, estimated_cost_micros_per_message,
    daily_cost_limit_micros, monthly_cost_limit_micros
)
SELECT gen_random_uuid(), 'box', id, 100, 1000, 100, 100000, 10000000, 100000000
FROM boxes;

INSERT INTO messaging_policies (
    id, scope, daily_message_limit, monthly_message_limit, per_dispatch_limit,
    estimated_cost_micros_per_message, daily_cost_limit_micros,
    monthly_cost_limit_micros
) VALUES (
    gen_random_uuid(), 'platform', 1000, 10000, 250, 100000, 100000000, 1000000000
);

CREATE TABLE messaging_usage_buckets (
    id UUID PRIMARY KEY,
    scope VARCHAR(20) NOT NULL,
    box_id UUID REFERENCES boxes(id) ON DELETE CASCADE,
    period_type VARCHAR(20) NOT NULL,
    period_start DATE NOT NULL,
    reserved_count INTEGER NOT NULL DEFAULT 0,
    accepted_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    reserved_cost_micros BIGINT NOT NULL DEFAULT 0,
    estimated_cost_micros BIGINT NOT NULL DEFAULT 0,
    actual_cost_micros BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT messaging_usage_scope_check CHECK (scope IN ('box', 'platform')),
    CONSTRAINT messaging_usage_period_check CHECK (period_type IN ('daily', 'monthly')),
    CONSTRAINT messaging_usage_owner_check CHECK (
        (scope = 'box' AND box_id IS NOT NULL) OR
        (scope = 'platform' AND box_id IS NULL)
    )
);

CREATE UNIQUE INDEX idx_messaging_usage_box_period
    ON messaging_usage_buckets(box_id, period_type, period_start) WHERE scope = 'box';
CREATE UNIQUE INDEX idx_messaging_usage_platform_period
    ON messaging_usage_buckets(scope, period_type, period_start) WHERE scope = 'platform';

CREATE TABLE message_dispatches (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    requested_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    source_type VARCHAR(40) NOT NULL,
    source_id UUID,
    connection_mode VARCHAR(20) NOT NULL,
    recipients_total INTEGER NOT NULL,
    reserved_messages INTEGER NOT NULL DEFAULT 0,
    accepted_messages INTEGER NOT NULL DEFAULT 0,
    failed_messages INTEGER NOT NULL DEFAULT 0,
    estimated_cost_micros BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(20) NOT NULL,
    block_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT message_dispatches_connection_check CHECK (connection_mode IN ('platform', 'dedicated')),
    CONSTRAINT message_dispatches_status_check CHECK (status IN ('reserved', 'completed', 'blocked', 'failed'))
);

CREATE INDEX idx_message_dispatches_box_created ON message_dispatches(box_id, created_at DESC);
CREATE INDEX idx_message_dispatches_status ON message_dispatches(status);

ALTER TABLE message_recipients
    ADD COLUMN provider_message_sid VARCHAR(64),
    ADD COLUMN provider_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN dispatch_id UUID REFERENCES message_dispatches(id) ON DELETE SET NULL;

ALTER TABLE workout_message_recipients
    ADD COLUMN provider_message_sid VARCHAR(64),
    ADD COLUMN provider_status VARCHAR(32) NOT NULL DEFAULT '',
    ADD COLUMN dispatch_id UUID REFERENCES message_dispatches(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX idx_message_recipients_provider_sid
    ON message_recipients(provider_message_sid) WHERE provider_message_sid IS NOT NULL;
CREATE UNIQUE INDEX idx_workout_recipients_provider_sid
    ON workout_message_recipients(provider_message_sid) WHERE provider_message_sid IS NOT NULL;

CREATE TABLE admin_audit_logs (
    id UUID PRIMARY KEY,
    admin_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(80) NOT NULL,
    target_type VARCHAR(40) NOT NULL,
    target_id VARCHAR(80) NOT NULL,
    before_data JSONB,
    after_data JSONB,
    reason TEXT NOT NULL DEFAULT '',
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_audit_logs_created ON admin_audit_logs(created_at DESC);
CREATE INDEX idx_admin_audit_logs_target ON admin_audit_logs(target_type, target_id);
