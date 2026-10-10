"""Outbox notifications for the invoice worker.

No mail is ever sent and no SMTP connection is ever opened: a notification row
is inserted into the outbox table so a separate process could pick it up later.
The row carries the recipient address, a subject, the message body and the
creation time, and starts in the status ``offen``.
"""

from decimal import Decimal
from typing import Any

from worker.invoice import Invoice

# German product copy (the customer portal speaks German).
SUBJECT_TEMPLATE = "Rechnung zum Auftrag {order_number}"
BODY_TEMPLATE = (
    "Guten Tag,\n\n"
    "zum Auftrag {order_number} wurde eine Rechnung erstellt.\n\n"
    "Zwischensumme: {subtotal} EUR\n"
    "MwSt. (19 %): {tax} EUR\n"
    "Gesamtbetrag: {total} EUR\n\n"
    "Mit freundlichen Grüßen\n"
    "Ihre Kfz-Werkstatt\n"
)

# The status a freshly queued notification starts in.
STATUS_OPEN = "offen"


def _euro(cents: int) -> str:
    """Format a whole number of cents as a German-style euro amount."""
    amount = (Decimal(cents) / Decimal(100)).quantize(Decimal("0.01"))
    return f"{amount:.2f}".replace(".", ",")


def enqueue_notification(conn: Any, invoice: Invoice) -> None:
    """Store a notification for the given invoice in the outbox.

    The caller owns the surrounding transaction; this function only writes the
    row, so an invoice and its notification are committed or rolled back
    together.
    """
    subject = SUBJECT_TEMPLATE.format(order_number=invoice.order_number)
    body = BODY_TEMPLATE.format(
        order_number=invoice.order_number,
        subtotal=_euro(invoice.subtotal_cents),
        tax=_euro(invoice.tax_cents),
        total=_euro(invoice.total_cents),
    )
    with conn.cursor() as cur:
        cur.execute(
            "INSERT INTO outbox_messages (recipient, subject, body, status)"
            " VALUES (%s, %s, %s, %s)",
            (invoice.recipient, subject, body, STATUS_OPEN),
        )
