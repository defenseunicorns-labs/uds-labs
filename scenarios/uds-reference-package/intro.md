# UDS Reference Package

The [UDS Reference Package](https://github.com/uds-packages/reference-package) is a maintained starting point for understanding how UDS-compatible packages are structured. It demonstrates Istio ambient mesh, Keycloak SSO, Postgres via an operator, Prometheus monitoring, and UDS policy enforcement.

In this lab you'll explore the package structure, understand why each layer exists where it does, deploy the full stack with `uds run dev`, and verify that the UDS Operator wired up the mesh, SSO, and monitoring from a single `Package` CR. Treat the repository as a living example, not a rule that every implementation detail must copy exactly.

**What's already running:** UDS Core (Keycloak, Istio, Pepr) on a k3d cluster.  
**What you'll deploy:** `postgres-operator` + `reference-package`, wired together via a UDS bundle.

> The image cache was pre-warmed during lab initialization. The terminal is ready now — you can start while setup finishes in the background.
