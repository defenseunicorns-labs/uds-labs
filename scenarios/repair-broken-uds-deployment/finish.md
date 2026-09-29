# Lab Complete

You repaired a package without changing the application source or patching the live cluster.

The defects crossed several boundaries:

- The non-root workload needed a writable temporary path and deliberate filesystem permissions.
- Resource settings needed to come from values rather than an oversized hard-coded profile.
- The Service, container port, UDS exposure, and uptime check needed one consistent contract.
- The Package CR network selector needed to match the actual workload labels.
- The package had to be rebuilt before the bundle could deploy the repaired source.

Remember the artifact vocabulary:

| Term | Meaning |
| --- | --- |
| Zarf package | Atomic deployable `.tar.zst` artifact containing package resources and embedded images. |
| UDS package | A Zarf package with UDS integration, including a Package CR. |
| Package CR | Kubernetes resource declaring network, exposure, SSO, monitoring, and policy needs. |
| UDS bundle | Deployable composition of packages with deployment-time configuration. |

The next production-shaped scenarios can extend this application with operator-managed PostgreSQL,
object storage, SSO, and lifecycle verification.
