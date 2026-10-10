"""Worker configuration, read lazily from the environment.

Nothing here is evaluated at import time: the values are read when
:func:`load_config` is called, so an incomplete environment fails with a message
that names the missing variable instead of a bare traceback at import.

``DATABASE_URL`` and ``VALKEY_URL`` are required — without them the worker cannot
do its job and must refuse to start. ``HOURLY_RATE_CENTS`` is declared
``external`` in RUN.json: the product must boot without it, so it is optional and
defaults to ``0`` (labour positions that carry no rate snapshot then bill at zero
cents per hour).
"""

import os
from dataclasses import dataclass


class MissingConfigError(RuntimeError):
    """A required configuration variable is absent or unusable."""


@dataclass(frozen=True)
class Config:
    """Resolved worker configuration."""

    database_url: str
    valkey_url: str
    hourly_rate_cents: int


def _required(name: str) -> str:
    value = os.environ.get(name)
    if value is None or value.strip() == "":
        raise MissingConfigError(
            f"missing required configuration variable {name} (see RUN.json for how the "
            f"worker is started and which values it needs)"
        )
    return value


def _optional_rate(name: str) -> int:
    value = os.environ.get(name)
    if value is None or value.strip() == "":
        return 0
    try:
        return int(value)
    except ValueError as exc:
        raise MissingConfigError(
            f"configuration variable {name} must be an integer number of cents, got {value!r}"
        ) from exc


def load_config() -> Config:
    """Read and validate the worker configuration from the environment.

    Raises:
        MissingConfigError: a required variable is missing, or HOURLY_RATE_CENTS
            is set but not an integer.
    """
    return Config(
        database_url=_required("DATABASE_URL"),
        valkey_url=_required("VALKEY_URL"),
        hourly_rate_cents=_optional_rate("HOURLY_RATE_CENTS"),
    )
