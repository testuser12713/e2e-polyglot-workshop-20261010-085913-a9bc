"""Invoice worker for the Kfz workshop portal.

The worker consumes invoice requests from the Valkey stream, builds an invoice
from the order's positions and stores it in PostgreSQL.
"""
