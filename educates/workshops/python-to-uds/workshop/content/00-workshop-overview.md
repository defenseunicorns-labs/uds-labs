# Package a Stateless App for UDS

This is the first vertical slice of the UDS Labs migration to Educates. It proves that an Educates session can provide the UDS authoring toolchain, a private Docker daemon, scoped Kubernetes access, and an arm64-native build environment without KubeVirt or CDI.

## What you will do

1. Confirm the entire session is running natively on arm64.
2. Inspect the preinstalled UDS CLI and Kubernetes access.
3. Build the sample Flask application as an arm64 container image.

> **Spike boundary:** This slice intentionally stops before Zarf package creation and deployment to shared UDS Core. Those are the next integration checkpoints after the arm64 runtime is proven.
