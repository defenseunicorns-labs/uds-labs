# Add Durable Object Storage to Release Ledger

This lab starts from the PostgreSQL-shaped Release Ledger contract. Release metadata belongs in PostgreSQL; uploaded bytes belong in an S3-compatible object store.

Configure a scoped MinIO app through the bundle, consume its copied Secret through `secretKeyRef`, and keep endpoint/bucket values non-secret. All work lives in `/root/release-ledger-minio`; do not edit `app.py` or patch live resources.
