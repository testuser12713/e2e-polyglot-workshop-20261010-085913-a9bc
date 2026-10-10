"""Tests for the reliable Valkey stream consumption in ``worker.queue``.

They run against the real Valkey addressed by ``VALKEY_URL`` (the same variable
the worker reads), with a test handler of their own. Each test uses its own
uniquely named stream so tests cannot interfere with each other or with the
product stream.
"""

from collections.abc import Iterator
from typing import Any
from uuid import uuid4

import pytest

from worker.queue import MAX_DELIVERIES, Queue

GROUP = "test-invoice-workers"
PAYLOAD = {"order_number": "A-1001", "plate": "B-XY 123"}


@pytest.fixture
def queue(valkey_url: str) -> Iterator[Queue]:
    """A Queue on its own throwaway stream, cleaned up afterwards."""
    suffix = uuid4().hex
    stream = f"test:invoices:{suffix}"
    dead_letter_stream = f"{stream}:dead"
    instance = Queue(
        valkey_url,
        stream=stream,
        group=GROUP,
        consumer="consumer-a",
        dead_letter_stream=dead_letter_stream,
        claim_min_idle_ms=0,
    )
    _drop(instance, stream, dead_letter_stream)
    yield instance
    _drop(instance, stream, dead_letter_stream)


def _drop(instance: Queue, stream: str, dead_letter_stream: str) -> None:
    instance.client.delete(stream, dead_letter_stream)


def test_ensure_group_is_idempotent(queue: Queue) -> None:
    """Creating the group twice is tolerated (the second answers BUSYGROUP)."""
    queue.ensure_group()
    queue.ensure_group()


def test_successful_message_is_acknowledged(queue: Queue) -> None:
    queue.ensure_group()
    queue.client.xadd(queue.stream, PAYLOAD)

    messages = queue.read_new(count=10)
    assert len(messages) == 1
    message_id, payload = messages[0]

    handled: list[dict[str, Any]] = []
    queue.process(lambda body: handled.append(body), message_id, payload)

    assert handled == [PAYLOAD]
    assert queue.client.xpending(queue.stream, queue.group)["pending"] == 0


def test_pending_message_is_picked_up_again(queue: Queue) -> None:
    """A message whose consumer died before acking is reclaimed by another."""
    queue.ensure_group()
    queue.client.xadd(queue.stream, PAYLOAD)

    delivered = queue.read_new(count=10)
    assert len(delivered) == 1
    message_id, payload = delivered[0]
    # The first consumer "crashed" here without acknowledging.
    assert queue.client.xpending(queue.stream, queue.group)["pending"] == 1

    survivor = Queue(
        queue.url,
        stream=queue.stream,
        group=queue.group,
        consumer="consumer-b",
        dead_letter_stream=queue.dead_letter_stream,
    )
    reclaimed = survivor.claim(min_idle_ms=0, count=10)

    assert reclaimed == [(message_id, payload)]
    handled: list[dict[str, Any]] = []
    survivor.process(lambda body: handled.append(body), reclaimed[0][0], reclaimed[0][1])

    assert handled == [PAYLOAD]
    assert survivor.client.xpending(queue.stream, queue.group)["pending"] == 0


def test_permanently_failing_message_lands_in_dead_letter(queue: Queue) -> None:
    queue.ensure_group()
    queue.client.xadd(queue.stream, PAYLOAD)

    attempts = 0

    def always_fails(payload: dict[str, Any]) -> None:
        nonlocal attempts
        attempts += 1
        raise RuntimeError("handler cannot process this message")

    delivered = queue.read_new(count=10)
    assert len(delivered) == 1
    queue.process(always_fails, *delivered[0])

    # Each round reclaims the still-pending message, until it exhausts the
    # allowed number of deliveries.
    while queue.client.xpending(queue.stream, queue.group)["pending"]:
        reclaimed = queue.claim(min_idle_ms=0, count=10)
        if not reclaimed:
            break
        queue.process(always_fails, *reclaimed[0])

    assert attempts == MAX_DELIVERIES
    assert queue.client.xpending(queue.stream, queue.group)["pending"] == 0

    dead = queue.client.xrange(queue.dead_letter_stream)
    assert len(dead) == 1
    dead_payload = dead[0][1]
    assert dead_payload["order_number"] == PAYLOAD["order_number"]
    assert dead_payload["plate"] == PAYLOAD["plate"]


def test_dead_letter_stream_is_not_the_product_stream() -> None:
    """Guard: the default dead-letter stream is the agreed interface name."""
    from worker.queue import INVOICE_DEAD_LETTER_STREAM

    assert INVOICE_DEAD_LETTER_STREAM == "workshop:invoices:dead"
