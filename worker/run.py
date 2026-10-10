"""Launcher: start the worker package from inside the worker directory.

``worker/queue.py`` shares its name with the standard library's ``queue`` module.
Keeping the worker directory on ``sys.path`` ahead of the standard library
breaks every dependency that imports ``queue`` (psycopg, redis), so this
launcher drops the worker directory and imports the package from the repository
root instead.
"""

import importlib
import os
import sys

if __name__ == "__main__":
    _here = os.path.dirname(os.path.abspath(__file__))
    _parent = os.path.dirname(_here)
    sys.path[:] = [p for p in sys.path if p and os.path.abspath(p) != _here]
    if _parent not in sys.path:
        sys.path.insert(0, _parent)
    raise SystemExit(importlib.import_module("worker.main").main())
