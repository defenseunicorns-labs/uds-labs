# Lab Complete

Release Ledger now separates durable state correctly:

- PostgreSQL stores release and evidence metadata.
- MinIO stores evidence bytes in the `evidence` bucket.
- The application uses scoped, copied credentials through Secret references.
- Network access is explicit on both application and dependency sides.
- Resource and storage settings are visible and environment-specific.

A production environment can replace these in-cluster dependencies without changing the application image or source.
