"""Outbox notifications for the invoice worker.

No mail is ever sent: a notification row is inserted into the outbox table so a
separate process could pick it up later.
"""

from typing import Any

from worker.invoice import Invoice


def enqueue_notification(conn: Any, invoice: Invoice) -> None:
    """Store a notification for the given invoice in the outbox."""
