"""Tests for the outbox notification of the invoice worker.

They run against the real PostgreSQL from ``DATABASE_URL`` and assert that a
notification row appears with recipient, subject, body and time and starts in
the status ``offen``. No mail is sent anywhere — the outbox is the only effect.
"""

import inspect

import psycopg

from worker import outbox
from worker.invoice import Invoice, InvoiceItem


def _invoice(recipient: str) -> Invoice:
    return Invoice(
        order_number="A-2026-0042",
        items=[
            InvoiceItem(
                kind="part",
                description="Bremsbelag",
                hours=None,
                quantity=1,
                unit_price_cents=21000,
                line_total_cents=21000,
            )
        ],
        subtotal_cents=21000,
        tax_cents=3990,
        total_cents=24990,
        recipient=recipient,
    )


def test_enqueue_notification_stores_a_row(
    schema_applied: psycopg.Connection, postgres_connection: psycopg.Connection
) -> None:
    suffix = "outbox-test@example.test"
    with postgres_connection.cursor() as cur:
        cur.execute("DELETE FROM outbox_messages WHERE recipient = %s", (suffix,))

    result = outbox.enqueue_notification(postgres_connection, _invoice(suffix))

    assert result is None
    with postgres_connection.cursor() as cur:
        cur.execute(
            "SELECT recipient, subject, body, status, created_at FROM outbox_messages"
            " WHERE recipient = %s",
            (suffix,),
        )
        rows = cur.fetchall()

    assert len(rows) == 1
    recipient, subject, body, status, created_at = rows[0]
    assert recipient == suffix
    assert "A-2026-0042" in subject
    assert "A-2026-0042" in body
    assert "249,90" in body
    assert status == "offen"
    assert created_at is not None

    with postgres_connection.cursor() as cur:
        cur.execute("DELETE FROM outbox_messages WHERE recipient = %s", (suffix,))
    postgres_connection.commit()


def test_open_status_is_the_agreed_literal() -> None:
    assert outbox.STATUS_OPEN == "offen"


def test_outbox_never_uses_smtp() -> None:
    """AC-10: no SMTP access exists in the notification path."""
    source = inspect.getsource(outbox)
    assert "smtplib" not in source
    assert "smtpd" not in source
