"""Shared fixtures for the worker tests.

The worker runs against a real PostgreSQL and a real Valkey, addressed through
the same environment variables the worker itself reads (DATABASE_URL,
VALKEY_URL). There is no SQLite and no in-memory queue substitute anywhere.
"""

import os
from pathlib import Path

import psycopg
import pytest
import redis

REPO_ROOT = Path(__file__).resolve().parents[2]
MIGRATION_FILE = REPO_ROOT / "backend" / "migrations" / "0001_init.sql"


@pytest.fixture(scope="session")
def database_url() -> str:
    """The PostgreSQL DSN from the environment."""
    url = os.environ.get("DATABASE_URL")
    if not url:
        pytest.fail("DATABASE_URL is not set; the worker tests need a real PostgreSQL")
    return url


@pytest.fixture(scope="session")
def valkey_url() -> str:
    """The Valkey address from the environment."""
    url = os.environ.get("VALKEY_URL")
    if not url:
        pytest.fail("VALKEY_URL is not set; the worker tests need a real Valkey")
    return url


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
def schema_applied(postgres_connection: psycopg.Connection) -> psycopg.Connection:
    """Apply the shared schema migration to the real database.

    Worker tests that need tables request this fixture. The migration file is
    owned by the API skeleton; if it is not present on the branch yet, the test
    is skipped rather than falling back to an in-memory schema.
    """
    if not MIGRATION_FILE.exists():
        pytest.skip(f"schema migration {MIGRATION_FILE} is not present on this branch")
    with postgres_connection.cursor() as cur:
        cur.execute(MIGRATION_FILE.read_text(encoding="utf-8"))
    postgres_connection.commit()
    return postgres_connection
