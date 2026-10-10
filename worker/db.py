"""PostgreSQL access for the worker."""

import psycopg


def connect(database_url: str) -> psycopg.Connection:
    """Open a PostgreSQL connection from the given DSN.

    The connection is returned open; callers own its lifecycle and transaction
    management.
    """
    return psycopg.connect(database_url)
