"""Shared fixtures for the worker tests.

The worker runs against a real PostgreSQL and a real Valkey, addressed through
the same environment variables the worker itself reads (DATABASE_URL,
VALKEY_URL). There is no SQLite and no in-memory queue substitute anywhere: a
missing database or queue makes the tests fail loudly instead of silently
switching to a stand-in.
"""

import os
from pathlib import Path

import psycopg
import pytest
import redis

REPO_ROOT = Path(__file__).resolve().parents[2]
MIGRATION_FILE = REPO_ROOT / "backend" / "migrations" / "0001_init.sql"


def _require_env(name: str, what: str) -> str:
    value = os.environ.get(name)
    if not value:
        pytest.fail(f"{name} is not set; the worker tests need a real {what}")
    return value


@pytest.fixture(scope="session")
def database_url() -> str:
    """The PostgreSQL DSN from the environment."""
    return _require_env("DATABASE_URL", "PostgreSQL")


@pytest.fixture(scope="session")
def valkey_url() -> str:
    """The Valkey address from the environment."""
    return _require_env("VALKEY_URL", "Valkey")


@pytest.fixture(scope="session")
def _schema_once(database_url: str) -> None:
    """Apply the shared schema migration to the real database exactly once.

    The migration is idempotent (``CREATE TABLE IF NOT EXISTS``) and lives with
    the API; the worker tests apply the very same file so they exercise the real
    schema.
    """
    with psycopg.connect(database_url) as conn, conn.cursor() as cur:
        cur.execute(MIGRATION_FILE.read_text(encoding="utf-8"))
        conn.commit()


@pytest.fixture
def pg_conn(database_url: str, _schema_once: None) -> psycopg.Connection:
    """An open connection to the real PostgreSQL, rolled back after each test."""
    with psycopg.connect(database_url) as conn:
        yield conn
        conn.rollback()


@pytest.fixture(scope="session")
def valkey_client(valkey_url: str) -> redis.Redis:
    """A real Valkey client built from VALKEY_URL."""
    client = redis.Redis.from_url(valkey_url, decode_responses=True)
    yield client
    client.close()


def test_data_stores_are_reachable(pg_conn: psycopg.Connection, valkey_client: redis.Redis) -> None:
    """Both declared stores answer: PostgreSQL to ``SELECT 1``, Valkey to ``PING``."""
    with pg_conn.cursor() as cur:
        cur.execute("SELECT 1")
        assert cur.fetchone() == (1,)
    assert valkey_client.ping() is True


# --- fixtures the worker's own feature modules already share -------------------


@pytest.fixture
def postgres_connection(database_url: str) -> psycopg.Connection:
    """An open connection to the real PostgreSQL, rolled back after each test."""
    with psycopg.connect(database_url) as conn:
        yield conn
        conn.rollback()


@pytest.fixture(scope="session")
def valkey(valkey_url: str) -> redis.Redis:
    """A real Valkey client built from VALKEY_URL."""
    client = redis.Redis.from_url(valkey_url, decode_responses=True)
    yield client
    client.close()


@pytest.fixture(scope="session")
def schema_connection(database_url: str) -> psycopg.Connection:
    """A session-scoped connection used only to apply the schema once."""
    with psycopg.connect(database_url) as conn:
        yield conn


@pytest.fixture(scope="session")
def schema_applied(schema_connection: psycopg.Connection) -> psycopg.Connection:
    """Apply the shared schema migration to the real database, once per session."""
    with schema_connection.cursor() as cur:
        cur.execute(MIGRATION_FILE.read_text(encoding="utf-8"))
    schema_connection.commit()
    return schema_connection
