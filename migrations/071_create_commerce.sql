CREATE TABLE commerce_products (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    name VARCHAR(120) NOT NULL,
    sku VARCHAR(64),
    price_cents INTEGER NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id, unit_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT commerce_products_name_check CHECK (char_length(btrim(name)) BETWEEN 2 AND 120),
    CONSTRAINT commerce_products_sku_check CHECK (sku IS NULL OR char_length(btrim(sku)) BETWEEN 1 AND 64),
    CONSTRAINT commerce_products_price_check CHECK (price_cents > 0),
    CONSTRAINT commerce_products_status_check CHECK (status IN ('active','inactive'))
);
CREATE UNIQUE INDEX commerce_products_sku_unique ON commerce_products(organization_id, unit_id, lower(sku)) WHERE sku IS NOT NULL;
CREATE INDEX commerce_products_unit_status ON commerce_products(unit_id, status, name);
CREATE TRIGGER set_commerce_products_updated_at BEFORE UPDATE ON commerce_products FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE commerce_sales (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    person_id UUID,
    status VARCHAR(16) NOT NULL DEFAULT 'completed',
    total_cents INTEGER NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'BRL',
    note VARCHAR(500),
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    sold_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (id, organization_id, unit_id),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT,
    FOREIGN KEY (person_id, organization_id) REFERENCES persons(id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT commerce_sales_status_check CHECK (status = 'completed'),
    CONSTRAINT commerce_sales_total_check CHECK (total_cents > 0),
    CONSTRAINT commerce_sales_currency_check CHECK (currency = 'BRL'),
    CONSTRAINT commerce_sales_note_check CHECK (note IS NULL OR char_length(btrim(note)) BETWEEN 1 AND 500)
);
CREATE INDEX commerce_sales_unit_sold_at ON commerce_sales(unit_id, sold_at DESC);
CREATE INDEX commerce_sales_person ON commerce_sales(person_id, sold_at DESC) WHERE person_id IS NOT NULL;

CREATE TABLE commerce_sale_items (
    id UUID PRIMARY KEY,
    sale_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    product_id UUID NOT NULL,
    product_name VARCHAR(120) NOT NULL,
    sku VARCHAR(64),
    quantity INTEGER NOT NULL,
    unit_price_cents INTEGER NOT NULL,
    total_cents INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (sale_id, organization_id, unit_id) REFERENCES commerce_sales(id, organization_id, unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (product_id, organization_id, unit_id) REFERENCES commerce_products(id, organization_id, unit_id) ON DELETE RESTRICT,
    CONSTRAINT commerce_sale_items_quantity_check CHECK (quantity BETWEEN 1 AND 999),
    CONSTRAINT commerce_sale_items_price_check CHECK (unit_price_cents > 0 AND total_cents = quantity * unit_price_cents),
    CONSTRAINT commerce_sale_items_product_unique UNIQUE (sale_id, product_id)
);
CREATE INDEX commerce_sale_items_sale ON commerce_sale_items(sale_id);

CREATE TABLE commerce_sale_payments (
    id UUID PRIMARY KEY,
    sale_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    unit_id UUID NOT NULL,
    method VARCHAR(24) NOT NULL,
    amount_cents INTEGER NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    actor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reference VARCHAR(120) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (sale_id, organization_id, unit_id) REFERENCES commerce_sales(id, organization_id, unit_id) ON DELETE RESTRICT,
    CONSTRAINT commerce_sale_payments_method_check CHECK (method IN ('pix','cash','card_machine')),
    CONSTRAINT commerce_sale_payments_amount_check CHECK (amount_cents > 0),
    CONSTRAINT commerce_sale_payments_one_per_sale UNIQUE (sale_id)
);

CREATE TABLE commerce_command_requests (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    command VARCHAR(32) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    fingerprint CHAR(64) NOT NULL,
    result_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (organization_id, unit_id, command, idempotency_key),
    FOREIGN KEY (unit_id, organization_id) REFERENCES boxes(id, organization_id) ON DELETE RESTRICT
);

CREATE OR REPLACE FUNCTION enforce_commerce_sale_totals()
RETURNS TRIGGER AS $$
DECLARE
    target_sale_id UUID;
    expected_total INTEGER;
    item_total BIGINT;
    payment_total BIGINT;
    payment_count INTEGER;
BEGIN
    IF TG_TABLE_NAME = 'commerce_sales' THEN
        target_sale_id := COALESCE(NEW.id, OLD.id);
    ELSE
        target_sale_id := COALESCE(NEW.sale_id, OLD.sale_id);
    END IF;
    SELECT total_cents INTO expected_total FROM commerce_sales WHERE id = target_sale_id;
    IF expected_total IS NULL THEN RETURN NULL; END IF;
    SELECT COALESCE(SUM(total_cents), 0) INTO item_total FROM commerce_sale_items WHERE sale_id = target_sale_id;
    SELECT COALESCE(SUM(amount_cents), 0), COUNT(*) INTO payment_total, payment_count FROM commerce_sale_payments WHERE sale_id = target_sale_id;
    IF item_total <> expected_total OR payment_total <> expected_total OR payment_count <> 1 THEN
        RAISE EXCEPTION 'commerce sale totals do not reconcile' USING ERRCODE = '23514';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER commerce_sales_totals_check
AFTER INSERT OR UPDATE ON commerce_sales DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_commerce_sale_totals();
CREATE CONSTRAINT TRIGGER commerce_sale_items_totals_check
AFTER INSERT OR UPDATE OR DELETE ON commerce_sale_items DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_commerce_sale_totals();
CREATE CONSTRAINT TRIGGER commerce_sale_payments_totals_check
AFTER INSERT OR UPDATE OR DELETE ON commerce_sale_payments DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW EXECUTE FUNCTION enforce_commerce_sale_totals();
