"""Invoice model and calculation.

The worker turns one finished order into one invoice: the labour positions bill
their hours at the rate snapshot stored on the position (or, when no snapshot is
available, the configured hourly rate), the part positions bill quantity times
unit price, 19 % VAT is added on the subtotal and every amount is a whole number
of cents. All arithmetic uses :class:`decimal.Decimal` with half-up rounding; no
cent amount is ever represented as a float.
"""

from dataclasses import dataclass, field
from datetime import datetime
from decimal import ROUND_HALF_UP, Decimal
from typing import Any

# The VAT rate applied to the subtotal. Kept as an exact Decimal so the tax is
# rounded half up on whole cents rather than drifting through binary floats.
VAT_RATE = Decimal("0.19")

# One cent, the quantum every monetary result is rounded to.
CENT = Decimal(1)


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
    # The customer e-mail the outbox notification is addressed to. It does not
    # belong on the API's invoice representation but the notification needs it.
    recipient: str = ""


def _to_decimal(value: Any) -> Decimal:
    """Return ``value`` as an exact Decimal, tolerating None, int, float, str."""
    if value is None:
        return Decimal(0)
    if isinstance(value, Decimal):
        return value
    return Decimal(str(value))


def _to_cents(value: Any) -> int:
    """Return ``value`` as a whole number of cents."""
    if value is None:
        return 0
    if isinstance(value, int):
        return value
    return int(_to_decimal(value))


def _round_half_up(value: Decimal) -> int:
    """Round a Decimal to whole cents, halves away from zero."""
    return int(value.quantize(CENT, rounding=ROUND_HALF_UP))


def _build_line(raw: dict[str, Any], hourly_rate_cents: int) -> InvoiceItem:
    """Compute one invoice position from a stored order item."""
    kind = str(raw.get("kind") or "")
    description = str(raw.get("description") or "")
    unit_price_cents = _to_cents(raw.get("unit_price_cents"))

    if kind == "labour":
        hours = _to_decimal(raw.get("hours"))
        # The rate snapshot on the position wins; a missing snapshot (0) falls
        # back to the configured hourly rate.
        rate = unit_price_cents if unit_price_cents > 0 else int(hourly_rate_cents)
        line_total_cents = _round_half_up(hours * Decimal(rate))
        return InvoiceItem(
            kind="labour",
            description=description,
            hours=float(hours),
            quantity=None,
            unit_price_cents=rate,
            line_total_cents=line_total_cents,
        )

    quantity = _to_cents(raw.get("quantity"))
    return InvoiceItem(
        kind=kind or "part",
        description=description,
        hours=None,
        quantity=quantity,
        unit_price_cents=unit_price_cents,
        line_total_cents=quantity * unit_price_cents,
    )


def build_invoice(
    order: dict[str, Any],
    items: list[dict[str, Any]],
    hourly_rate_cents: int,
) -> Invoice:
    """Build an :class:`Invoice` from an order and its positions.

    Args:
        order: the order row (at least ``order_number``; ``customer_email``
            when the notification recipient is to be filled in).
        items: the order's positions as stored in ``order_items``.
        hourly_rate_cents: the configured rate used when a labour position has
            no rate snapshot of its own.
    """
    invoice_items = [_build_line(raw, hourly_rate_cents) for raw in items]
    subtotal_cents = sum(item.line_total_cents for item in invoice_items)
    tax_cents = _round_half_up(Decimal(subtotal_cents) * VAT_RATE)
    return Invoice(
        order_number=str(order.get("order_number") or ""),
        items=invoice_items,
        subtotal_cents=subtotal_cents,
        tax_cents=tax_cents,
        total_cents=subtotal_cents + tax_cents,
        recipient=str(order.get("customer_email") or order.get("email") or ""),
    )
