# Lab Complete

Release Ledger now uses operator-managed PostgreSQL for release metadata.

You practiced the production boundary:

- The bundle composes and configures the dependency.
- The operator creates credentials in the consuming namespace.
- ConfigMaps carry non-secret endpoint and database settings.
- Secret references carry generated username and password values.
- Application egress and database ingress form a reciprocal network contract.
- A real write/read surviving pod replacement verifies the result.

The next scenario adds MinIO-backed evidence content while retaining this PostgreSQL baseline.
