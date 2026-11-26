# Educates platform UDS package

Standalone UDS/Zarf package for the official Educates `4.0.0-alpha.10` Helm runtime. It installs the upstream session and secrets managers plus a small local UDS Core integration chart. It does not install Contour, Kyverno, or the Educates CLI.

## Create

```bash
uds run educates:create-platform-package
uds run educates:create-platform-bundle
```

These create the arm64 Zarf package under this directory and a standalone UDS bundle under `bundle/`.

## Deploy independently

```bash
uds deploy \
  educates/packages/platform/bundle/uds-bundle-educates-platform-arm64-4.0.0-alpha.10-uds.0.tar.zst \
  --set DOMAIN=uds.dev \
  --confirm
```

The target cluster must run Kubernetes 1.31 or newer and have UDS Core base, a tenant gateway, a Zarf registry, and a default StorageClass.

## Package boundary

The platform owns shared Educates CRDs and controllers. Workshop packages own their `Workshop`, `TrainingPortal`, portal route, dynamic session routes, workshop image, and optional workload images.

See:

- [`docs/configuration.md`](docs/configuration.md)
- [`docs/upstream.md`](docs/upstream.md)
- [`docs/justifications.md`](docs/justifications.md)

## Qualification boundary

The source is statically rendered and linted, but live arm64 deployment, UDS package validation, dynamic workshop routing, removal, and upgrade behavior remain qualification gates. Educates 4.0.0-alpha.10 is a prerelease.
