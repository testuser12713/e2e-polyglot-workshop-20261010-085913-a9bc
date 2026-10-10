"""Worker test package.

`worker/queue.py` shares its name with the standard library's ``queue`` module.
The CI runs pytest from inside ``worker/`` with ``PYTHONPATH=.`` which would put
``worker/queue.py`` ahead of the stdlib and break every dependency that imports
``queue`` (psycopg, redis). Keep the repository root on the path so the
``worker`` package imports, and drop the worker directory itself so the real
``queue`` module wins.
"""

import importlib
import importlib.util
import os
import subprocess
import sys

_REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
_WORKER_DIR = os.path.join(_REPO_ROOT, "worker")
_REQUIREMENTS = os.path.join(_WORKER_DIR, "requirements.txt")

sys.path[:] = [p for p in sys.path if p and os.path.abspath(p) != _WORKER_DIR]
if _REPO_ROOT not in sys.path:
    sys.path.insert(0, _REPO_ROOT)


def _ensure_dependencies() -> None:
    """Install the declared requirements when a test dependency is missing.

    The pipeline installs the worker manifest before running the suite, so this
    is a no-op there. On a bare interpreter that skipped the install it keeps
    the suite from aborting at collection with an ImportError, by installing the
    declared manifest once before the test modules import psycopg and redis.
    """
    missing = [name for name in ("psycopg", "redis") if importlib.util.find_spec(name) is None]
    if not missing or not os.path.isfile(_REQUIREMENTS):
        return
    subprocess.run(
        [sys.executable, "-m", "pip", "install", "-q", "-r", _REQUIREMENTS],
        check=True,
    )
    importlib.invalidate_caches()


_ensure_dependencies()
