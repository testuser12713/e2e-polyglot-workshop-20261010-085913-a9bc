"""Valkey stream queue used by the worker.

The stream, consumer group and consumer names are fixed by the sprint's shared
interface. This module owns the delivery mechanics of reliable consumption:

* :meth:`Queue.ensure_group` creates the group and the stream together, and is
  idempotent (a concurrent worker creating it first answers ``BUSYGROUP``).
* :meth:`Queue.run` is the blocking consume loop; every message is handed to a
  handler and only acknowledged after the handler returned successfully.
* :meth:`Queue.claim` reclaims messages whose consumer died before acking them,
  so an unacknowledged message is processed again after a restart.
* A permanently failing message is retried up to ``max_deliveries`` times and
  then moved to the dead-letter stream instead of looping forever.

The module is deliberately free of invoice arithmetic: it only moves and
accounts messages.
"""

import logging
from collections.abc import Callable
from typing import Any

import redis
from redis.exceptions import ResponseError

INVOICE_STREAM = "workshop:invoices"
INVOICE_GROUP = "invoice-workers"
INVOICE_DEAD_LETTER_STREAM = "workshop:invoices:dead"

# How many times one message may be delivered before it is dead-lettered.
MAX_DELIVERIES = 3

# Default block time of the read loop, in milliseconds.
DEFAULT_READ_BLOCK_MS = 5000

# The socket read timeout must be longer than the server-side block time of a
# blocking XREADGROUP, otherwise an idle stream raises a socket TimeoutError
# instead of the read returning empty after read_block_ms. redis-py's own
# default socket timeout is only 5s, so it is set explicitly here.
SOCKET_TIMEOUT_MARGIN_S = 5.0

# Default idle time before a pending message is considered abandoned by a dead
# consumer and reclaimed, in milliseconds.
DEFAULT_CLAIM_MIN_IDLE_MS = 60_000

logger = logging.getLogger(__name__)

Message = tuple[str, dict[str, Any]]


class Queue:
    """A Valkey stream consumer bound to one stream, group and consumer name."""

    def __init__(
        self,
        url: str,
        stream: str,
        group: str,
        consumer: str,
        dead_letter_stream: str = INVOICE_DEAD_LETTER_STREAM,
        max_deliveries: int = MAX_DELIVERIES,
        read_count: int = 10,
        read_block_ms: int = DEFAULT_READ_BLOCK_MS,
        claim_min_idle_ms: int = DEFAULT_CLAIM_MIN_IDLE_MS,
    ) -> None:
        self.url = url
        self.stream = stream
        self.group = group
        self.consumer = consumer
        self.dead_letter_stream = dead_letter_stream
        self.max_deliveries = max_deliveries
        self.read_count = read_count
        self.read_block_ms = read_block_ms
        self.claim_min_idle_ms = claim_min_idle_ms
        # Keep the socket read timeout strictly longer than the blocking read's
        # block time, so idling in the consume loop is not mistaken for a dead
        # connection. A block time of 0 means "block forever", in which case the
        # socket timeout is disabled as well.
        if read_block_ms and read_block_ms > 0:
            socket_timeout: float | None = read_block_ms / 1000.0 + SOCKET_TIMEOUT_MARGIN_S
        else:
            socket_timeout = None
        self.client = redis.Redis.from_url(
            url,
            decode_responses=True,
            socket_timeout=socket_timeout,
        )
        # Touch the server once so a worker started against an unreachable Valkey
        # fails at startup instead of silently doing nothing.
        self.client.ping()

    def ensure_group(self) -> None:
        """Create the consumer group on the stream if it does not exist yet.

        ``MKSTREAM`` creates the stream as well, so the first worker to start
        does not need another process to have written to it. Creating a group
        that already exists answers ``BUSYGROUP``, which is the expected outcome
        when a second worker starts and is tolerated here.
        """
        try:
            self.client.xgroup_create(self.stream, self.group, id="0", mkstream=True)
        except ResponseError as exc:
            if "BUSYGROUP" not in str(exc):
                raise

    def run(self, handler: Callable[[dict[str, Any]], None]) -> None:
        """Consume messages and dispatch each payload to ``handler``.

        The loop never returns on its own. On startup it takes over the pending
        messages a previous (dead) consumer left behind, then blocks on new
        messages while reclaiming anything idle beyond ``claim_min_idle_ms``. A
        message is acknowledged only after its handler succeeded; failures are
        retried until they exhaust ``max_deliveries`` and are dead-lettered.
        """
        self.ensure_group()
        # The consumer name is per process, so at startup every pending entry
        # belongs to a consumer that is gone: take them over at once, without
        # waiting for the idle timeout, so a restart reprocesses them.
        for message_id, payload in self.claim(min_idle_ms=0):
            self.process(handler, message_id, payload)
        while True:
            for message_id, payload in self.claim():
                self.process(handler, message_id, payload)
            for message_id, payload in self.read_new(block_ms=self.read_block_ms):
                self.process(handler, message_id, payload)

    def read_new(self, count: int | None = None, block_ms: int | None = None) -> list[Message]:
        """Read messages never delivered to this group (the ``>`` cursor).

        Args:
            count: maximum number of messages to read.
            block_ms: how long to wait for a message; ``None`` returns at once.
        """
        if count is None:
            count = self.read_count
        response = self.client.xreadgroup(
            self.group,
            self.consumer,
            {self.stream: ">"},
            count=count,
            block=block_ms,
        )
        if not response:
            return []
        _, messages = response[0]
        return [(message_id, fields) for message_id, fields in messages]

    def claim(
        self,
        min_idle_ms: int | None = None,
        count: int | None = None,
        start_id: str = "0-0",
    ) -> list[Message]:
        """Reclaim pending messages whose consumer died before acking them.

        Uses ``XAUTOCLAIM`` so a restarted worker takes over the messages that
        its predecessor left pending, so nothing is lost.
        """
        if min_idle_ms is None:
            min_idle_ms = self.claim_min_idle_ms
        if count is None:
            count = self.read_count
        result = self.client.xautoclaim(
            self.stream,
            self.group,
            self.consumer,
            min_idle_time=min_idle_ms,
            start_id=start_id,
            count=count,
        )
        messages = result[1] if len(result) >= 2 else []
        return [(message_id, fields) for message_id, fields in messages]

    def process(
        self,
        handler: Callable[[dict[str, Any]], None],
        message_id: str,
        payload: dict[str, Any],
    ) -> None:
        """Dispatch one message and account for its outcome.

        On success the message is acknowledged. On failure it stays pending so
        it is reclaimed and retried, until it has been delivered
        ``max_deliveries`` times, at which point it is moved to the dead-letter
        stream and removed from the pending list.
        """
        try:
            handler(payload)
        except Exception:
            attempts = self.delivery_count(message_id)
            if attempts >= self.max_deliveries:
                self.to_dead_letter(message_id, payload)
            else:
                logger.warning(
                    "message %s failed on delivery %d/%d; it will be retried",
                    message_id,
                    attempts,
                    self.max_deliveries,
                )
        else:
            self.ack(message_id)

    def delivery_count(self, message_id: str) -> int:
        """Return how often ``message_id`` has been delivered so far.

        The count lives in Valkey's own pending-entries table; it is 1 for a
        freshly read message and grows with every reclaim.
        """
        entries = self.client.xpending_range(self.stream, self.group, message_id, message_id, 1)
        if not entries:
            return 0
        return int(entries[0]["times_delivered"])

    def ack(self, message_id: str) -> int:
        """Acknowledge a message that was processed successfully.

        Called only after the handler returned without raising.
        """
        return int(self.client.xack(self.stream, self.group, message_id))

    def to_dead_letter(self, message_id: str, payload: dict[str, Any]) -> str:
        """Move a message that exhausted its retries to the dead-letter stream.

        The payload is copied to the dead-letter stream together with where it
        came from, then the original delivery is acknowledged so the loop does
        not run forever.
        """
        fields: dict[str, Any] = dict(payload)
        fields["original_stream"] = self.stream
        fields["original_message_id"] = message_id
        dead_letter_id = self.client.xadd(self.dead_letter_stream, fields)
        self.ack(message_id)
        logger.warning(
            "message %s exhausted %d deliveries, moved to %s",
            message_id,
            self.max_deliveries,
            self.dead_letter_stream,
        )
        return str(dead_letter_id)
