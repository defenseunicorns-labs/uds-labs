# Educates UDS package spike

This directory evaluates Educates as the arm64, cluster-native replacement for the KubeVirt/CDI UDS Labs runtime. Installation and workshop delivery use UDS/Zarf packages; consumers do not need the Educates CLI binary.

## Packages

- [`packages/platform`](packages/platform) — official Educates `4.0.0-alpha.10` Helm runtime and UDS Core integration.
- [`workshops/python-to-uds`](workshops/python-to-uds) — arm64 workshop image, `Workshop`, and `TrainingPortal`.
- [`bundle`](bundle) — installs the platform package before the workshop package.

The platform package is independently reusable. It contains no workshop content, Contour, or Kyverno.

## Build on Apple Silicon

```bash
uds run educates:validate
uds run educates:create-bundle
```

The package build downloads the pinned upstream arm64 images and may take several minutes. It produces:

```text
educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.3.2.tar.zst
```

## Deploy

Provision a disposable arm64 Kubernetes 1.31+ cluster with UDS Core base, a tenant gateway, a default RWO StorageClass, and DNS for the selected tenant domain. Then deploy:

```bash
uds deploy \
  educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.3.2.tar.zst \
  --set DOMAIN=uds.dev \
  --confirm
```

Verify the platform, portal route, and any allocated session routes without the Educates CLI:

```bash
uds run educates:verify-deployment
```

The portal is available at `https://uds-labs-ui.<tenant-domain>`. Allocate a session and rerun the verification task to check its dashboard and editor VirtualServices.

## Upstream runtime

The platform consumes the official OCI chart directly:

```text
oci://ghcr.io/educates/charts/educates-training-platform:4.0.0-alpha.10
```

The chart is digest-pinned in the Zarf package. The local Helm chart contains only UDS Core networking, controller RBAC, and an intentionally inactive IngressClass. Workshop packages must provide UDS routes for portals and sessions.

## Status

This remains a spike. The workshop now includes the reference implementation's portal and dynamic session routing model. Live arm64 deployment, privileged Docker policy behavior, session cleanup, package validation, and removal remain verification gates. Educates 4.0.0-alpha.10 is a prerelease.

See [`docs/research/2026-11-26-educates-arm64-platform-assessment.md`](../docs/research/2026-11-26-educates-arm64-platform-assessment.md) for the original architecture assessment.
