"""Tests for the worker configuration and its start-up contract.

The configuration is read lazily; the subprocess tests exercise the declared
entry point ``python -m worker.main`` from the repository root, so a missing
required variable really aborts the process with a message naming it.
"""

import os
import subprocess
import sys
from pathlib import Path

import pytest

from worker import config

REPO_ROOT = Path(__file__).resolve().parents[2]


def _clean_env(**overrides: str) -> dict[str, str]:
    """The current environment without the worker variables, plus ``overrides``."""
    env = dict(os.environ)
    for name in ("DATABASE_URL", "VALKEY_URL", "HOURLY_RATE_CENTS", "PYTHONPATH"):
        env.pop(name, None)
    env.update(overrides)
    return env


def test_load_config_reads_all_values(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
    monkeypatch.setenv("VALKEY_URL", "redis://localhost:6379/0")
    monkeypatch.setenv("HOURLY_RATE_CENTS", "8500")

    cfg = config.load_config()

    assert cfg.database_url == "postgresql://app@localhost:5432/app"
    assert cfg.valkey_url == "redis://localhost:6379/0"
    assert cfg.hourly_rate_cents == 8500


def test_hourly_rate_is_optional(monkeypatch: pytest.MonkeyPatch) -> None:
    """HOURLY_RATE_CENTS is external: the worker boots without it (rate 0)."""
    monkeypatch.setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
    monkeypatch.setenv("VALKEY_URL", "redis://localhost:6379/0")
    monkeypatch.delenv("HOURLY_RATE_CENTS", raising=False)

    cfg = config.load_config()

    assert cfg.hourly_rate_cents == 0


def test_missing_database_url_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("VALKEY_URL", "redis://localhost:6379/0")
    monkeypatch.delenv("DATABASE_URL", raising=False)

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "DATABASE_URL" in str(excinfo.value)


def test_missing_valkey_url_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
    monkeypatch.delenv("VALKEY_URL", raising=False)

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "VALKEY_URL" in str(excinfo.value)


def test_non_integer_hourly_rate_names_the_variable(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("DATABASE_URL", "postgresql://app@localhost:5432/app")
    monkeypatch.setenv("VALKEY_URL", "redis://localhost:6379/0")
    monkeypatch.setenv("HOURLY_RATE_CENTS", "not-a-number")

    with pytest.raises(config.MissingConfigError) as excinfo:
        config.load_config()

    assert "HOURLY_RATE_CENTS" in str(excinfo.value)


def test_main_exits_non_zero_without_database_url() -> None:
    env = _clean_env(VALKEY_URL="redis://localhost:6379/0")

    result = subprocess.run(
        [sys.executable, "-m", "worker.main"],
        cwd=REPO_ROOT,
        env=env,
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode != 0
    assert "DATABASE_URL" in result.stderr


def test_main_exits_non_zero_without_valkey_url() -> None:
    env = _clean_env(DATABASE_URL="postgresql://app@localhost:5432/app")

    result = subprocess.run(
        [sys.executable, "-m", "worker.main"],
        cwd=REPO_ROOT,
        env=env,
        capture_output=True,
        text=True,
        timeout=30,
    )

    assert result.returncode != 0
    assert "VALKEY_URL" in result.stderr
