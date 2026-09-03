# Lab Complete

You packaged and deployed Release Ledger without adding a database or object store. The application worked, but its Release data disappeared when Kubernetes replaced the pod.

The important boundary is now clear: the application package owns the workload and UDS integration, while the bundle is the place to compose shared backing services.

The next production-shaped scenario moves Release metadata into operator-managed PostgreSQL. A later scenario moves evidence content into MinIO so it survives application pod replacement.
