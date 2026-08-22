# Lab Complete

You've deployed the UDS Reference Package end-to-end and seen every major UDS integration in action:

- **Zarf** scoped the application package to the application and its UDS integration, not shared infrastructure dependencies.
- **The bundle** composed `postgres-operator` and `reference-package`, with environment-specific configuration at the deployment layer.
- **The UDS Package CR** declared the app's exposure, network, SSO, and monitoring needs; the UDS Operator generated the supporting platform resources.
- **The Postgres adapter** solved a connection-string compatibility problem, while also showing why render-time Secret lookup can race operator reconciliation.
- **Ambient mesh** enforced mTLS and captured metrics at the node level via ztunnel — no sidecar injection required.

Carry the boundaries forward rather than copying every line: application and UDS integration in the application package, shared dependencies in the bundle, direct Secret references where the application supports them, and platform needs in the UDS Package CR.

**Next:** Try *Package a Stateless App for UDS* — start from a Python Flask app and author each package layer.
