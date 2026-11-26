# Package a Stateless App for UDS — Educates workshop package

This is the workshop half of the Educates arm64 spike. Content is embedded in its arm64 container image. A separate Zarf package installs the `Workshop`, `TrainingPortal`, portal UDS integration, and dynamic per-session UDS routes. Consumers do not install or execute the Educates CLI.

## What it proves

- native arm64 Educates workshop container
- pinned UDS CLI `v0.35.1`
- private per-session Docker daemon
- Educates-managed Kubernetes credentials
- UDS tenant-gateway routes for the portal, dashboard, and editor
- arm64 sample application build
- examiner checks for the runtime and application image

The slice intentionally stops before deploying a learner-created Zarf package to shared UDS Core.

## Validate

```bash
uds run educates:validate
```

Validation renders the content through the pinned Educates 3.8 CLI container because Educates 4.0 is prerelease-only and has no equivalent stable CLI release. Runtime resources and CRDs target the pinned Educates 4.0 Helm platform. No host Educates binary is required.

## Build and package

On Apple Silicon:

```bash
uds run educates:build-image
uds run educates:create-workshop-package
```

The default image is `ghcr.io/defenseunicorns-labs/educates-python-to-uds-image:0.1.0` for `linux/arm64` and remains in the local Docker daemon so Zarf can package it. Normally use `uds run educates:create-bundle` to build the platform and workshop together.

## Runtime and routing

The portal is `https://uds-labs-ui.<tenant-domain>`. Educates creates the portal and environment namespaces. The workshop definition creates UDS Packages and an internal Istio ServiceEntry for every session, producing tenant VirtualServices for the dashboard and editor without an additional ingress controller.

Each session requests:

- 2 GiB workshop memory and 5 GiB storage
- 2 GiB Docker sidecar memory and 10 GiB Docker storage
- a `large` Educates namespace budget

Docker support is privileged. Deploy only to a disposable, dedicated cluster until the security model has completed live validation.

See [`docs/configuration.md`](docs/configuration.md) and [`docs/justifications.md`](docs/justifications.md).
