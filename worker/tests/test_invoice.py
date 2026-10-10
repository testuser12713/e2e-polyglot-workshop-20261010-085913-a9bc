"""Tests for the invoice calculation and the message handler.

The calculation is checked with plain values; the handler is exercised against
the real PostgreSQL that ``DATABASE_URL`` points at, using the shared schema from
``backend/migrations/0001_init.sql``. Each test seeds its own order and removes
exactly the rows it created.
"""

import logging
from collections.abc import Iterator
from decimal import Decimal
from typing import Any
from uuid import uuid4

import psycopg
import pytest

from worker.handler import handle_message
from worker.invoice import build_invoice

HOURLY_RATE_CENTS = 9000


@pytest.fixture(autouse=True)
def _configured_hourly_rate(monkeypatch: pytest.MonkeyPatch) -> None:
    """Give the handler the rate the worker is configured with in production."""
    monkeypatch.setenv("HOURLY_RATE_CENTS", str(HOURLY_RATE_CENTS))


def test_labour_and_part_amounts_include_tax() -> None:
    order = {"order_number": "A-1", "customer_email": "kunde@example.test"}
    items = [
        {
            "kind": "labour",
            "description": "Bremsen",
            "hours": Decimal("2.50"),
            "quantity": None,
            "unit_price_cents": 8500,
            "line_total_cents": 0,
        },
        {
            "kind": "part",
            "description": "Belag",
            "hours": None,
            "quantity": 3,
            "unit_price_cents": 1250,
            "line_total_cents": 0,
        },
    ]

    invoice = build_invoice(order, items, HOURLY_RATE_CENTS)

    assert [i.line_total_cents for i in invoice.items] == [21250, 3750]
    assert invoice.subtotal_cents == 25000
    assert invoice.tax_cents == 4750
    assert invoice.total_cents == 29750
    assert invoice.recipient == "kunde@example.test"


def test_labour_without_snapshot_uses_configured_rate() -> None:
    items = [
        {
            "kind": "labour",
            "description": "Arbeit",
            "hours": Decimal("1.00"),
            "quantity": None,
            "unit_price_cents": 0,
            "line_total_cents": 0,
        }
    ]

    invoice = build_invoice({"order_number": "A-2"}, items, HOURLY_RATE_CENTS)

    assert invoice.items[0].unit_price_cents == HOURLY_RATE_CENTS
    assert invoice.items[0].line_total_cents == 9000
    assert invoice.tax_cents == 1710
    assert invoice.total_cents == 10710


def test_labour_line_total_rounds_half_up() -> None:
    items = [
        {
            "kind": "labour",
            "description": "Arbeit",
            "hours": Decimal("0.50"),
            "quantity": None,
            "unit_price_cents": 3333,
            "line_total_cents": 0,
        },
        {
            "kind": "labour",
            "description": "Arbeit",
            "hours": Decimal("1.50"),
            "quantity": None,
            "unit_price_cents": 3333,
            "line_total_cents": 0,
        },
    ]

    invoice = build_invoice({"order_number": "A-3"}, items, HOURLY_RATE_CENTS)

    # 0.5 * 3333 = 1666.5 -> 1667, 1.5 * 3333 = 4999.5 -> 5000 (half up).
    assert [i.line_total_cents for i in invoice.items] == [1667, 5000]


def test_tax_rounds_half_up_on_whole_cents() -> None:
    items = [
        {
            "kind": "part",
            "description": "Kleinigkeit",
            "hours": None,
            "quantity": 1,
            "unit_price_cents": 3,
            "line_total_cents": 0,
        }
    ]

    invoice = build_invoice({"order_number": "A-4"}, items, HOURLY_RATE_CENTS)

    assert invoice.subtotal_cents == 3
    # 3 * 0.19 = 0.57 -> 1 cent (half up).
    assert invoice.tax_cents == 1
    assert invoice.total_cents == 4


def test_invoice_without_items_has_zero_amounts() -> None:
    invoice = build_invoice({"order_number": "A-5"}, [], HOURLY_RATE_CENTS)

    assert invoice.items == []
    assert (invoice.subtotal_cents, invoice.tax_cents, invoice.total_cents) == (0, 0, 0)


@pytest.fixture
def seeded_order(
    schema_applied: psycopg.Connection, postgres_connection: psycopg.Connection
) -> Iterator[dict[str, Any]]:
    """Insert one finished order with a labour and a part position, then clean up."""
    suffix = uuid4().hex[:12]
    order_number = f"A-TEST-{suffix}"
    email = f"kunde-{suffix}@example.test"
    plate = f"T-{suffix[:6]}"

    cur = postgres_connection.cursor()
    cur.execute(
        "INSERT INTO customers (name, email, phone) VALUES (%s, %s, %s) RETURNING id",
        ("Test Kunde", email, "0000"),
    )
    customer_id = cur.fetchone()[0]
    cur.execute(
        "INSERT INTO vehicles (plate, plate_normalized, brand, model, mileage)"
        " VALUES (%s, %s, %s, %s, %s) RETURNING id",
        (plate, plate.upper(), "VW", "Golf", 1000),
    )
    vehicle_id = cur.fetchone()[0]
    cur.execute(
        "INSERT INTO workshop_orders (order_number, customer_id, vehicle_id, status, description)"
        " VALUES (%s, %s, %s, %s, %s) RETURNING id",
        (order_number, customer_id, vehicle_id, "fertig", "Bremsen"),
    )
    order_id = cur.fetchone()[0]
    cur.execute(
        "INSERT INTO order_items (order_id, kind, description, hours, unit_price_cents, line_total_cents)"
        " VALUES (%s, %s, %s, %s, %s, %s)",
        (order_id, "labour", "Arbeitszeit", Decimal("2.50"), 8500, 21250),
    )
    cur.execute(
        "INSERT INTO order_items (order_id, kind, description, quantity, unit_price_cents, line_total_cents)"
        " VALUES (%s, %s, %s, %s, %s, %s)",
        (order_id, "part", "Bremsbelag", 3, 1250, 3750),
    )
    postgres_connection.commit()

    yield {"order_number": order_number, "email": email, "order_id": order_id}

    with postgres_connection.cursor() as cleanup:
        cleanup.execute("DELETE FROM outbox_messages WHERE recipient = %s", (email,))
        cleanup.execute("DELETE FROM invoices WHERE order_number = %s", (order_number,))
        cleanup.execute("DELETE FROM order_items WHERE order_id = %s", (order_id,))
        cleanup.execute("DELETE FROM workshop_orders WHERE id = %s", (order_id,))
        cleanup.execute("DELETE FROM vehicles WHERE id = %s", (vehicle_id,))
        cleanup.execute("DELETE FROM customers WHERE id = %s", (customer_id,))
    postgres_connection.commit()


def test_handle_message_persists_invoice_and_outbox(
    seeded_order: dict[str, Any], postgres_connection: psycopg.Connection
) -> None:
    handle_message(
        {"order_number": seeded_order["order_number"], "plate": "IGNORED"}, postgres_connection
    )

    with postgres_connection.cursor() as cur:
        cur.execute(
            "SELECT subtotal_cents, tax_cents, total_cents FROM invoices WHERE order_number = %s",
            (seeded_order["order_number"],),
        )
        invoice = cur.fetchone()
        assert invoice == (25000, 4750, 29750)

        cur.execute(
            "SELECT i.kind, i.description, i.line_total_cents"
            " FROM invoice_items i JOIN invoices v ON v.id = i.invoice_id"
            " WHERE v.order_number = %s ORDER BY i.id",
            (seeded_order["order_number"],),
        )
        assert cur.fetchall() == [("labour", "Arbeitszeit", 21250), ("part", "Bremsbelag", 3750)]

        cur.execute(
            "SELECT recipient, subject, body, status, created_at FROM outbox_messages"
            " WHERE recipient = %s",
            (seeded_order["email"],),
        )
        outbox = cur.fetchall()
        assert len(outbox) == 1
        recipient, subject, body, status, created_at = outbox[0]
        assert recipient == seeded_order["email"]
        assert seeded_order["order_number"] in subject
        assert seeded_order["order_number"] in body
        assert status == "offen"
        assert created_at is not None


def test_handle_message_is_idempotent(
    seeded_order: dict[str, Any], postgres_connection: psycopg.Connection
) -> None:
    message = {"order_number": seeded_order["order_number"], "plate": "IGNORED"}
    handle_message(message, postgres_connection)
    handle_message(message, postgres_connection)

    with postgres_connection.cursor() as cur:
        cur.execute(
            "SELECT count(*) FROM invoices WHERE order_number = %s",
            (seeded_order["order_number"],),
        )
        assert cur.fetchone()[0] == 1

        cur.execute(
            "SELECT count(*) FROM invoice_items i JOIN invoices v ON v.id = i.invoice_id"
            " WHERE v.order_number = %s",
            (seeded_order["order_number"],),
        )
        assert cur.fetchone()[0] == 2

        cur.execute(
            "SELECT count(*) FROM outbox_messages WHERE recipient = %s",
            (seeded_order["email"],),
        )
        assert cur.fetchone()[0] == 1


def test_handle_message_logs_no_personal_data(
    seeded_order: dict[str, Any],
    postgres_connection: psycopg.Connection,
    caplog: pytest.LogCaptureFixture,
) -> None:
    with caplog.at_level(logging.INFO):
        handle_message({"order_number": seeded_order["order_number"]}, postgres_connection)

    assert seeded_order["email"] not in caplog.text


def test_handle_message_rejects_missing_order_number(
    postgres_connection: psycopg.Connection,
) -> None:
    with pytest.raises(ValueError):
        handle_message({}, postgres_connection)
