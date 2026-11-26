# Decisions and applicability

- Educates retains native authentication. The platform controllers expose no end-user endpoint, so this package does not create an SSO client.
- Portal and session exposure belongs to separately packaged workshops; the platform `Package` CR declares controller networking only.
- No verified Prometheus application endpoint exists for the enabled managers, so the package does not create a monitor entry.
- Educates 4.0.0-alpha.10 lacks Kopf CRD discovery permission. A local read-only role grants `get`, `list`, and `watch` for CRDs to the managers.
- The trusted session manager may create UDS `Package` and Istio `ServiceEntry` resources supplied by workshop object templates. These permissions are not granted to learners.
- Upstream Ingress resources use the controllerless `educates-unused` class. UDS workshop packages must provide external routes.
- This package uses a prerelease chart and is not presented as a production-ready Educates 4 release.
- Upgrade, registry publication, disconnected deployment, and live arm64 UDS Core qualification remain unverified.

## Validator applicability

- `sso-presence`: native Educates authentication is an explicit spike decision; platform controllers have no login endpoint.
- `monitor-presence`: no verified metrics endpoint exists for the enabled controller profile.
