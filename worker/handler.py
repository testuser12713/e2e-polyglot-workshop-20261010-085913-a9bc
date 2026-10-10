"""Message handler for invoices consumed from the Valkey stream.

The queue delivers one ``{order_number, plate}`` message per finished order. This
handler reads the order with its positions from PostgreSQL, builds the invoice,
persists it idempotently and enqueues the outbox notification — all in one
transaction, so a crash leaves either a complete invoice with its notification
or nothing at all.

No personal data (customer name, e-mail, phone or plate) is ever logged; only the
non-personal order number appears in log output.
"""

import logging
from typing import Any

from psycopg.rows import dict_row

from worker import config
from worker.invoice import build_invoice
from worker.outbox import enqueue_notification

logger = logging.getLogger(__name__)

_SELECT_ORDER = """
    SELECT o.id AS order_id,
           o.order_number,
           c.email AS customer_email
      FROM workshop_orders o
      JOIN customers c ON c.id = o.customer_id
     WHERE o.order_number = %s
"""

_SELECT_ITEMS = """
    SELECT kind, description, hours, quantity, unit_price_cents, line_total_cents
      FROM order_items
     WHERE order_id = %s
     ORDER BY id
"""

_INSERT_INVOICE = """
    INSERT INTO invoices (order_number, order_id, subtotal_cents, tax_cents, total_cents)
         VALUES (%s, %s, %s, %s, %s)
    ON CONFLICT (order_number) DO NOTHING
      RETURNING id, created_at
"""

_INSERT_INVOICE_ITEM = """
    INSERT INTO invoice_items
           (invoice_id, kind, description, hours, quantity, unit_price_cents, line_total_cents)
    VALUES (%s, %s, %s, %s, %s, %s, %s)
"""


def handle_message(payload: dict[str, Any], conn: Any) -> None:
    """Process one queue message: build, persist and announce the invoice."""
    order_number = payload.get("order_number") if isinstance(payload, dict) else None
    if not order_number:
        raise ValueError("invoice message is missing order_number")

    hourly_rate_cents = config.load_config().hourly_rate_cents

    with conn.transaction(), conn.cursor(row_factory=dict_row) as cur:
        cur.execute(_SELECT_ORDER, (order_number,))
        order = cur.fetchone()
        if order is None:
            raise LookupError(f"order {order_number} does not exist")

        cur.execute(_SELECT_ITEMS, (order["order_id"],))
        items = cur.fetchall()

        invoice = build_invoice(order, items, hourly_rate_cents)

        cur.execute(
            _INSERT_INVOICE,
            (
                invoice.order_number,
                order["order_id"],
                invoice.subtotal_cents,
                invoice.tax_cents,
                invoice.total_cents,
            ),
        )
        inserted = cur.fetchone()

        if inserted is None:
            # AC-09: the invoice already exists, so neither its positions
            # nor the outbox notification are written a second time.
            logger.info("invoice for order %s already exists; skipping", order_number)
            return

        invoice.created_at = inserted["created_at"]
        for item in invoice.items:
            cur.execute(
                _INSERT_INVOICE_ITEM,
                (
                    inserted["id"],
                    item.kind,
                    item.description,
                    item.hours,
                    item.quantity,
                    item.unit_price_cents,
                    item.line_total_cents,
                ),
            )

        enqueue_notification(conn, invoice)

    logger.info("created invoice for order %s", order_number)
