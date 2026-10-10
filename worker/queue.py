"""Valkey stream queue used by the worker.

The stream, consumer group and consumer names are fixed by the sprint's shared
interface. The delivery mechanics (reliable consumption with retries and a
dead-letter stream) are provided by a follow-up ticket; only the signatures are
fixed here.
"""

from collections.abc import Callable
from typing import Any

import redis

INVOICE_STREAM = "workshop:invoices"
INVOICE_GROUP = "invoice-workers"


class Queue:
    """A Valkey stream consumer bound to one stream, group and consumer name."""

    def __init__(self, url: str, stream: str, group: str, consumer: str) -> None:
        self.url = url
        self.stream = stream
        self.group = group
        self.consumer = consumer
        self.client = redis.Redis.from_url(url, decode_responses=True)
        # Touch the server once so a worker started against an unreachable Valkey
        # fails at startup instead of silently doing nothing.
        self.client.ping()

    def ensure_group(self) -> None:
        """Create the consumer group on the stream if it does not exist yet."""

    def run(self, handler: Callable[[dict[str, Any]], None]) -> None:
        """Consume messages and dispatch each payload to ``handler``."""

    def claim(self, *args: Any, **kwargs: Any) -> Any:
        """Claim pending messages that belong to this consumer."""

    def ack(self, *args: Any, **kwargs: Any) -> Any:
        """Acknowledge a successfully processed message."""

    def to_dead_letter(self, *args: Any, **kwargs: Any) -> Any:
        """Move a message that exhausted its retries to the dead-letter stream."""
