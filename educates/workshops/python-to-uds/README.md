# Package a Stateless App for UDS — Educates workshop package

This is the workshop half of the Educates arm64 spike. The workshop content is embedded directly in its multi-architecture container image, and its `Workshop` and `TrainingPortal` resources are deployed by Helm from a Zarf package. Consumers do not install or execute the Educates CLI.

## What it proves

- native arm64 Educates workshop container
- pinned UDS CLI `v0.35.1`
- private per-session Docker daemon
- Educates-managed Kubernetes credentials
- arm64 sample application build
- examiner checks for the runtime and application image

The slice intentionally stops before deploying a Zarf package to shared UDS Core.

## Validate

Validation uses the official multi-architecture Educates CLI **container** to render the content, then lints the Helm chart and Zarf definition:

```bash
uds run educates:validate
```

No host Educates binary is required.

## Build the workshop image

On an Apple Silicon Mac:

```bash
uds run educates:build-image
```

The default build is `linux/arm64`, tagged as:

```text
ghcr.io/defenseunicorns-labs/educates-python-to-uds-image:0.1.0
```

The image stays in the local Docker daemon so Zarf can add it to the package. Set `push=1` only when publishing the standalone image is desired.

## Create the workshop package

```bash
uds run educates:create-workshop-package
```

Normally use `uds run educates:create-bundle` instead. It creates the separate platform package first and then combines both packages into one deployable bundle.

## Runtime resources

Each session requests:

- 2 GiB workshop memory
- 5 GiB workshop storage
- 2 GiB Docker sidecar memory
- 10 GiB Docker storage
- a `large` Educates namespace budget

Docker support is privileged. Deploy only to a disposable, dedicated cluster until the security model has completed live validation.
