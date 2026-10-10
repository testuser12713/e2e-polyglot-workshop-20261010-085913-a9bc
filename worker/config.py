"""Worker configuration, read lazily from the environment.

Nothing here is evaluated at import time: the values are read when
:func:`load_config` is called, so an incomplete environment fails with a message
that names the missing variable instead of a bare traceback at import.
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


def load_config() -> Config:
    """Read and validate the worker configuration from the environment.

    Raises:
        MissingConfigError: a required variable is missing or not an integer.
    """
    database_url = _required("DATABASE_URL")
    valkey_url = _required("VALKEY_URL")
    raw_rate = _required("HOURLY_RATE_CENTS")
    try:
        hourly_rate_cents = int(raw_rate)
    except ValueError as exc:
        raise MissingConfigError(
            f"configuration variable HOURLY_RATE_CENTS must be an integer number of cents, "
            f"got {raw_rate!r}"
        ) from exc
    return Config(
        database_url=database_url,
        valkey_url=valkey_url,
        hourly_rate_cents=hourly_rate_cents,
    )
