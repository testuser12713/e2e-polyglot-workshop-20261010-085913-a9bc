"""Worker test package.

`worker/queue.py` shares its name with the standard library's ``queue`` module.
The CI runs pytest from inside ``worker/`` with ``PYTHONPATH=.`` which would put
``worker/queue.py`` ahead of the stdlib and break every dependency that imports
``queue`` (psycopg, redis). Keep the repository root on the path so the
``worker`` package imports, and drop the worker directory itself so the real
``queue`` module wins.
"""

import os
import sys

_REPO_ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
_WORKER_DIR = os.path.join(_REPO_ROOT, "worker")

sys.path[:] = [p for p in sys.path if p and os.path.abspath(p) != _WORKER_DIR]
if _REPO_ROOT not in sys.path:
    sys.path.insert(0, _REPO_ROOT)
