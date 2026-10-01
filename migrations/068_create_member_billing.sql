CREATE TABLE member_billing_contracts (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    billing_cycle VARCHAR(16) NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    currency CHAR(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    status VARCHAR(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active','canceled')),
    payment_method VARCHAR(16) NOT NULL CHECK (payment_method IN ('pix','boleto','card')),
    started_on DATE NOT NULL,
    next_due_on DATE,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id), UNIQUE (organization_id, enrollment_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (enrollment_id, organization_id) REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT
);

CREATE TABLE member_charges (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    person_id UUID NOT NULL,
    enrollment_id UUID NOT NULL,
    contract_id UUID NOT NULL,
    sequence INTEGER NOT NULL CHECK (sequence > 0),
    due_on DATE NOT NULL,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    paid_cents INTEGER NOT NULL DEFAULT 0 CHECK (paid_cents >= 0 AND paid_cents <= amount_cents),
    refunded_cents INTEGER NOT NULL DEFAULT 0 CHECK (refunded_cents >= 0 AND refunded_cents <= paid_cents),
    currency CHAR(3) NOT NULL DEFAULT 'BRL' CHECK (currency = 'BRL'),
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','partially_paid','paid','canceled','refunded')),
    payment_method VARCHAR(16) NOT NULL CHECK (payment_method IN ('pix','boleto','card')),
    provider VARCHAR(32) NOT NULL DEFAULT '', provider_reference VARCHAR(160) NOT NULL DEFAULT '',
    payment_instructions JSONB NOT NULL DEFAULT '{}'::jsonb,
    paid_at TIMESTAMPTZ, canceled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id), UNIQUE (organization_id, contract_id, sequence), UNIQUE (organization_id, provider, provider_reference),
    FOREIGN KEY (contract_id, organization_id) REFERENCES member_billing_contracts(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (enrollment_id, organization_id) REFERENCES enrollments(id, organization_id) ON DELETE RESTRICT
);
CREATE INDEX idx_member_charges_unit_due ON member_charges(organization_id, unit_id, due_on, status);
CREATE INDEX idx_member_charges_person ON member_charges(organization_id, person_id, due_on DESC);

CREATE TABLE member_payments (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    charge_id UUID NOT NULL,
    kind VARCHAR(24) NOT NULL CHECK (kind IN ('manual','provider_settlement','refund')),
    method VARCHAR(20) NOT NULL CHECK (method IN ('pix','cash','card_machine','provider','refund')),
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0), occurred_at TIMESTAMPTZ NOT NULL,
    actor_user_id UUID REFERENCES users(id) ON DELETE RESTRICT,
    provider_event_id VARCHAR(160) NOT NULL DEFAULT '', reason VARCHAR(240) NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id), FOREIGN KEY (charge_id, organization_id) REFERENCES member_charges(id, organization_id) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX idx_member_payments_provider_event ON member_payments(organization_id, provider_event_id) WHERE provider_event_id <> '';

CREATE TABLE member_billing_command_requests (
    id UUID PRIMARY KEY, organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    command VARCHAR(32) NOT NULL, idempotency_key VARCHAR(128) NOT NULL, request_fingerprint CHAR(64) NOT NULL, result_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), UNIQUE (organization_id, command, idempotency_key),
    CHECK (idempotency_key ~ '^[A-Za-z0-9._:-]{8,128}$'), CHECK (request_fingerprint ~ '^[0-9a-f]{64}$')
);
