# Educates platform spike

This directory contains throwaway migration work used to evaluate Educates as the arm64, cluster-native replacement for the KubeVirt/CDI UDS Labs runtime.

- [`workshops/python-to-uds`](workshops/python-to-uds): first vertical slice for the stateless UDS packaging scenario.
- [`docs/research/2026-11-26-educates-arm64-platform-assessment.md`](../docs/research/2026-11-26-educates-arm64-platform-assessment.md): architecture and arm64 assessment.

The spike is intentionally isolated from the production bundle and application. It should not be wired into the release path until runtime, security, ingress, and cleanup findings are validated on a disposable arm64 cluster.
