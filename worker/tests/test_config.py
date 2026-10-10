"""Tests for the worker configuration and for connectivity to its data stores."""

import psycopg
import pytest
import redis

from worker import config


def _set_valid_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
    monkeypatch.setenv("VALKEY_URL", "redis://localhost:6379/0")
    monkeypatch.setenv("HOURLY_RATE_CENTS", "8500")


def test_load_config_reads_all_values(monkeypatch: pytest.MonkeyPatch) -> None:
    _set_valid_env(monkeypatch)

    cfg = config.load_config()

    assert cfg.database_url == "postgresql://app@localhost:5432/app"
    assert cfg.valkey_url == "redis://localhost:6379/0"
    assert cfg.hourly_rate_cents == 8500


def test_missing_database_url_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    _set_valid_env(monkeypatch)
    monkeypatch.delenv("DATABASE_URL", raising=False)

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "DATABASE_URL" in str(excinfo.value)


def test_missing_valkey_url_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    _set_valid_env(monkeypatch)
    monkeypatch.delenv("VALKEY_URL", raising=False)

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "VALKEY_URL" in str(excinfo.value)


def test_missing_hourly_rate_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    _set_valid_env(monkeypatch)
    monkeypatch.delenv("HOURLY_RATE_CENTS", raising=False)

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "HOURLY_RATE_CENTS" in str(excinfo.value)


def test_non_integer_hourly_rate_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    _set_valid_env(monkeypatch)
    monkeypatch.setenv("HOURLY_RATE_CENTS", "not-a-number")

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "HOURLY_RATE_CENTS" in str(excinfo.value)


def test_postgres_and_valkey_are_reachable(
    postgres_connection: psycopg.Connection, valkey: redis.Redis
) -> None:
    """One connectivity test: both declared stores answer."""
    with postgres_connection.cursor() as cur:
        cur.execute("SELECT 1")
        assert cur.fetchone() == (1,)
    assert valkey.ping() is True
