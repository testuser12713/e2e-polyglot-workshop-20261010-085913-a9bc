"""Entry point of the invoice worker.

Start with ``python -m worker.main`` from the repository root. The worker loads
its configuration, connects to PostgreSQL and Valkey, ensures the consumer group
exists, announces readiness on stdout and then consumes the invoice stream.
"""

import os
import socket
import sys

from worker import config, db
from worker.handler import handle_message
from worker.queue import INVOICE_GROUP, INVOICE_STREAM, Queue


def main() -> int:
    """Run the worker. Returns the process exit code."""
    try:
        cfg = config.load_config()
    except config.MissingConfigError as exc:
        print(f"worker: {exc}", file=sys.stderr, flush=True)
        return 1

    conn = db.connect(cfg.database_url)
    consumer = f"{socket.gethostname()}-{os.getpid()}"
    queue = Queue(cfg.valkey_url, INVOICE_STREAM, INVOICE_GROUP, consumer)
    queue.ensure_group()

    print("worker ready", flush=True)
    try:
        queue.run(lambda payload: handle_message(payload, conn))
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
