-- Workshop portal schema. Idempotent: the API applies it at startup and the
-- worker's tests apply the same file.

CREATE TABLE IF NOT EXISTS customers (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT        NOT NULL,
    email      TEXT        NOT NULL,
    phone      TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS customers_email_uidx ON customers (email);

CREATE TABLE IF NOT EXISTS vehicles (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    plate            TEXT        NOT NULL,
    plate_normalized TEXT        NOT NULL,
    brand            TEXT        NOT NULL DEFAULT '',
    model            TEXT        NOT NULL DEFAULT '',
    mileage          INTEGER     NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS vehicles_plate_normalized_uidx ON vehicles (plate_normalized);

CREATE TABLE IF NOT EXISTS workshop_orders (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_number   TEXT        NOT NULL,
    customer_id    BIGINT      NOT NULL REFERENCES customers (id),
    vehicle_id     BIGINT      NOT NULL REFERENCES vehicles (id),
    status         TEXT        NOT NULL DEFAULT 'angefragt',
    preferred_date DATE,
    description    TEXT        NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS workshop_orders_order_number_uidx ON workshop_orders (order_number);

CREATE TABLE IF NOT EXISTS order_items (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id         BIGINT      NOT NULL REFERENCES workshop_orders (id) ON DELETE CASCADE,
    kind             TEXT        NOT NULL,
    description      TEXT        NOT NULL DEFAULT '',
    hours            NUMERIC(8, 2),
    quantity         INTEGER,
    unit_price_cents BIGINT,
    line_total_cents BIGINT      NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS order_status_history (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id   BIGINT      NOT NULL REFERENCES workshop_orders (id) ON DELETE CASCADE,
    status     TEXT        NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS employees (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS employees_email_uidx ON employees (email);

CREATE TABLE IF NOT EXISTS invoices (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_number   TEXT        NOT NULL,
    order_id       BIGINT      REFERENCES workshop_orders (id),
    subtotal_cents BIGINT      NOT NULL DEFAULT 0,
    tax_cents      BIGINT      NOT NULL DEFAULT 0,
    total_cents    BIGINT      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS invoices_order_number_uidx ON invoices (order_number);

CREATE TABLE IF NOT EXISTS invoice_items (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_id       BIGINT      NOT NULL REFERENCES invoices (id) ON DELETE CASCADE,
    kind             TEXT        NOT NULL,
    description      TEXT        NOT NULL DEFAULT '',
    hours            NUMERIC(8, 2),
    quantity         INTEGER,
    unit_price_cents BIGINT,
    line_total_cents BIGINT      NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS outbox_messages (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    recipient  TEXT        NOT NULL,
    subject    TEXT        NOT NULL,
    body       TEXT        NOT NULL,
    status     TEXT        NOT NULL DEFAULT 'offen',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
