"""Invoice model and calculation entry point.

The calculation itself (labour hours x hourly rate plus parts, 19 % VAT, whole
cents) is delivered by a follow-up ticket; the dataclasses and the signature are
fixed here so the worker wiring can already reference them.
"""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any


@dataclass
class InvoiceItem:
    """A single position on an invoice."""

    kind: str
    description: str
    hours: float | None
    quantity: int | None
    unit_price_cents: int
    line_total_cents: int


@dataclass
class Invoice:
    """A computed invoice for one workshop order."""

    order_number: str
    items: list[InvoiceItem] = field(default_factory=list)
    subtotal_cents: int = 0
    tax_cents: int = 0
    total_cents: int = 0
    created_at: datetime | None = None


def build_invoice(
    order: dict[str, Any],
    items: list[dict[str, Any]],
    hourly_rate_cents: int,
) -> Invoice:
    """Build an :class:`Invoice` from an order and its positions."""

    raise NotImplementedError("invoice calculation is implemented by the invoice ticket")
