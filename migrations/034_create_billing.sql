ALTER TABLE boxes
    ADD COLUMN billing_access_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN billing_access_reason TEXT NOT NULL DEFAULT '',
    ADD COLUMN billing_access_changed_at TIMESTAMPTZ;

CREATE TABLE billing_plans (
    id UUID PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    monthly_price_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'BRL',
    monthly_message_limit INTEGER NOT NULL,
    daily_message_limit INTEGER NOT NULL,
    per_dispatch_limit INTEGER NOT NULL,
    warning_percent INTEGER NOT NULL DEFAULT 80,
    grace_period_days INTEGER NOT NULL DEFAULT 7,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT billing_plans_code_version_unique UNIQUE (code, version),
    CONSTRAINT billing_plans_price_check CHECK (monthly_price_cents >= 0),
    CONSTRAINT billing_plans_currency_check CHECK (currency = 'BRL'),
    CONSTRAINT billing_plans_limits_check CHECK (
        monthly_message_limit >= 0
        AND daily_message_limit >= 0
        AND per_dispatch_limit >= 0
        AND daily_message_limit <= monthly_message_limit
        AND per_dispatch_limit <= daily_message_limit
    ),
    CONSTRAINT billing_plans_warning_check CHECK (warning_percent BETWEEN 1 AND 100),
    CONSTRAINT billing_plans_grace_check CHECK (grace_period_days BETWEEN 0 AND 90)
);

CREATE UNIQUE INDEX billing_plans_one_active_version
    ON billing_plans(code)
    WHERE active;

CREATE TABLE billing_customers (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL UNIQUE REFERENCES boxes(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL DEFAULT 'asaas',
    provider_customer_id VARCHAR(128) UNIQUE,
    legal_name VARCHAR(255) NOT NULL,
    cpf_cnpj VARCHAR(20) NOT NULL DEFAULT '',
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(32) NOT NULL DEFAULT '',
    postal_code VARCHAR(16) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    address_number VARCHAR(32) NOT NULL DEFAULT '',
    complement VARCHAR(255) NOT NULL DEFAULT '',
    province VARCHAR(128) NOT NULL DEFAULT '',
    city VARCHAR(128) NOT NULL DEFAULT '',
    state VARCHAR(2) NOT NULL DEFAULT '',
    notification_disabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT billing_customers_provider_check CHECK (provider = 'asaas')
);

CREATE TABLE billing_subscriptions (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    billing_customer_id UUID NOT NULL REFERENCES billing_customers(id) ON DELETE RESTRICT,
    plan_id UUID NOT NULL REFERENCES billing_plans(id) ON DELETE RESTRICT,
    provider VARCHAR(32) NOT NULL DEFAULT 'asaas',
    provider_subscription_id VARCHAR(128) UNIQUE,
    status VARCHAR(32) NOT NULL,
    billing_type VARCHAR(32) NOT NULL,
    next_due_date DATE NOT NULL,
    current_period_start DATE,
    current_period_end DATE,
    grace_until DATE,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    canceled_at TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    external_reference VARCHAR(128) NOT NULL UNIQUE,
    last_reconciled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT billing_subscriptions_provider_check CHECK (provider = 'asaas'),
    CONSTRAINT billing_subscriptions_status_check CHECK (
        status IN ('trialing', 'pending', 'active', 'past_due', 'suspended', 'canceled')
    ),
    CONSTRAINT billing_subscriptions_type_check CHECK (
        billing_type IN ('UNDEFINED', 'BOLETO', 'CREDIT_CARD', 'PIX')
    )
);

CREATE UNIQUE INDEX billing_subscriptions_one_current_per_box
    ON billing_subscriptions(box_id)
    WHERE status <> 'canceled';

CREATE INDEX billing_subscriptions_status_due_idx
    ON billing_subscriptions(status, next_due_date);

CREATE TABLE billing_invoices (
    id UUID PRIMARY KEY,
    box_id UUID NOT NULL REFERENCES boxes(id) ON DELETE CASCADE,
    subscription_id UUID NOT NULL REFERENCES billing_subscriptions(id) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL DEFAULT 'asaas',
    provider_payment_id VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(48) NOT NULL,
    billing_type VARCHAR(32) NOT NULL,
    value_cents BIGINT NOT NULL,
    net_value_cents BIGINT,
    due_date DATE NOT NULL,
    original_due_date DATE,
    confirmed_at TIMESTAMPTZ,
    received_at TIMESTAMPTZ,
    invoice_url TEXT NOT NULL DEFAULT '',
    bank_slip_url TEXT NOT NULL DEFAULT '',
    external_reference VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT billing_invoices_value_check CHECK (value_cents >= 0),
    CONSTRAINT billing_invoices_net_value_check CHECK (net_value_cents IS NULL OR net_value_cents >= 0)
);

CREATE INDEX billing_invoices_box_due_idx ON billing_invoices(box_id, due_date DESC);
CREATE INDEX billing_invoices_subscription_idx ON billing_invoices(subscription_id, due_date DESC);
CREATE INDEX billing_invoices_status_idx ON billing_invoices(status);

CREATE TABLE billing_webhook_events (
    id UUID PRIMARY KEY,
    provider VARCHAR(32) NOT NULL DEFAULT 'asaas',
    provider_event_id VARCHAR(160) NOT NULL,
    event_type VARCHAR(96) NOT NULL,
    provider_payment_id VARCHAR(128) NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status VARCHAR(24) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT billing_webhook_events_unique UNIQUE (provider, provider_event_id),
    CONSTRAINT billing_webhook_events_status_check CHECK (status IN ('pending', 'processed', 'failed'))
);

CREATE INDEX billing_webhook_events_status_idx
    ON billing_webhook_events(status, received_at);

INSERT INTO billing_plans (
    id, code, version, name, description, monthly_price_cents,
    monthly_message_limit, daily_message_limit, per_dispatch_limit,
    warning_percent, grace_period_days
) VALUES
(
    gen_random_uuid(), 'pilot_300', 1, 'Piloto 300',
    'Piloto comercial assistido de 90 dias com WhatsApp incluído.',
    29700, 300, 100, 100, 80, 7
),
(
    gen_random_uuid(), 'engagefit_500', 1, 'EngageFit 500',
    'Plano comercial por unidade com WhatsApp incluído.',
    39700, 500, 100, 100, 80, 7
);
