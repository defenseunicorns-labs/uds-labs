# Educates UDS package spike

This directory evaluates Educates as the arm64, cluster-native replacement for the KubeVirt/CDI UDS Labs runtime. Installation and workshop delivery use UDS/Zarf packages; consumers do not need the Educates CLI binary.

## Packages

- [`packages/platform`](packages/platform) — Educates 3.8.0, Contour, Kyverno, and Docker support.
- [`workshops/python-to-uds`](workshops/python-to-uds) — arm64 workshop image, `Workshop`, and `TrainingPortal`.
- [`bundle`](bundle) — installs the platform package before the workshop package.

## Build on Apple Silicon

```bash
uds run educates:validate
uds run educates:create-bundle
```

The package build downloads the pinned upstream arm64 images and may take several minutes. It produces:

```text
educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.1.0.tar.zst
```

## Deploy

Provision a disposable arm64 Kubernetes cluster with a default RWO StorageClass and LoadBalancer support. Configure wildcard DNS for the selected domain to point at the Contour LoadBalancer, then deploy:

```bash
uds deploy \
  educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.1.0.tar.zst \
  --set DOMAIN=educates.example.com \
  --confirm
```

Inspect the generated portal URL and credentials without the Educates CLI:

```bash
kubectl get trainingportals.training.educates.dev
kubectl describe trainingportal uds-labs
```

## Regenerating platform manifests

The platform package commits the output of Educates' official 3.8.0 renderer. Regeneration runs the upstream CLI as a container, not as a host binary:

```bash
uds run educates:render-platform
```

## Status

This remains a spike. Static package validation works, but live arm64 deployment, Pepr exemptions, Contour/Istio coexistence, Kyverno/Pepr interactions, wildcard routing, and cleanup still require validation before this can become a production package.

See [`docs/research/2026-11-26-educates-arm64-platform-assessment.md`](../docs/research/2026-11-26-educates-arm64-platform-assessment.md) for the architecture assessment.
