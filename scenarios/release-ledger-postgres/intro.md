# Connect Release Ledger to PostgreSQL

Release Ledger currently stores metadata in process memory. Your task is to connect the application to an operator-managed PostgreSQL instance without changing the image.

The application package owns the workload and its UDS Package CR. The bundle owns the PostgreSQL dependency and its environment-specific configuration. The operator creates the application role's credential Secret; Git should contain only its name and key names.

All work lives in `/root/release-ledger-postgres`. Do not edit `app.py` or patch the live cluster.
