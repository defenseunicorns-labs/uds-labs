# Decisions and applicability

- Educates native anonymous registration and session ownership remain enabled for this disposable-cluster spike. No Keycloak client or Authservice gate is created. This is an explicit exception to the standard UDS end-user SSO requirement.
- No verified Prometheus application endpoint exists for the portal or workshop runtime, so no monitor entry is created.
- Educates creates and owns the portal, environment, exercise, and session namespaces. The Zarf package waits for the portal namespace before installing its portal `Package` CR; it does not pre-create that namespace.
- One environment `Package` protects the shared backend namespace. Per-session routing namespaces contain separate `Package` resources whose selectors intentionally match no local pods; the internal `ServiceEntry` selects the actual backend pod in the environment namespace.
- An Istio `ServiceEntry` is used instead of an `ExternalName` Service because UDS policy rejects `ExternalName` Services.
- The trusted Educates session manager receives Package and ServiceEntry lifecycle permissions from the separately installed platform package. Learners do not receive those permissions or a RoleBinding in routing namespaces.
- Educates 4.0.0-alpha.10 starts Kopf inside each training portal but omits CRD discovery from the generated portal role. The workshop chart binds only its `training-portal` service account to the platform's read-only CRD-reader role (`get`, `list`, and `watch`). The deploy lifecycle restarts the portal after installing that binding because Kopf does not rebuild the failed discovery watch dynamically.
- Docker-in-Docker is privileged. This workshop is restricted to disposable, dedicated clusters pending live Pepr review and broader security qualification.
- No policy Exemption is included. If live deployment is denied, first determine whether the workshop can be changed to satisfy policy before considering an exemption.

## Validator applicability

- `sso-presence`: native Educates authentication is an explicit spike decision.
- `monitor-presence`: no verified metrics endpoint exists.
- `flavors`: the package has one workshop image and no alternate binary distribution.
- `ci-files`: this nested spike is built and validated through root repository tasks.
