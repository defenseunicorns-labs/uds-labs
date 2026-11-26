# Upstream provenance

- Project: [Educates Training Platform](https://github.com/educates/educates-training-platform)
- Chart: `oci://ghcr.io/educates/charts/educates-training-platform`
- Release: `4.0.0-alpha.10`
- Chart digest: `sha256:4db518658f044fea7aa5a968df6c2caadee94ec202819f1d092e57fe06c56757`
- Source revision: [`25dd76f65011b07b6eee2c65fe1d1bceb71f74c1`](https://github.com/educates/educates-training-platform/tree/25dd76f65011b07b6eee2c65fe1d1bceb71f74c1)
- License: Apache-2.0

The package consumes the official OCI chart directly and does not vendor or regenerate upstream manifests. The local chart contains only UDS Core integration, a controllerless IngressClass, and the alpha.10 RBAC corrections documented in [`justifications.md`](justifications.md).

Educates 4.0.0-alpha.10 requires Kubernetes 1.31 or newer and remains a prerelease. Update the chart digest, chart release, explicitly listed images, `appVersion`, and package version together.
