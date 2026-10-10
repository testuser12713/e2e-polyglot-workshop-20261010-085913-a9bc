"""Message handler for invoices consumed from the Valkey stream."""

from typing import Any


def handle_message(payload: dict[str, Any], conn: Any) -> None:
    """Process one queue message: build, persist and announce the invoice."""
