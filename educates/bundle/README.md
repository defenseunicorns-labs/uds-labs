# Educates UDS Labs spike bundle

The bundle installs two separate packages in order:

1. `educates` — the official Educates `4.0.0-alpha.10` Helm runtime and UDS Core integration.
2. `educates-python-to-uds` — the arm64 workshop image and its Educates resources.

Create both Zarf packages and the bundle with:

```bash
uds run educates:create-bundle
```

Deploy to a disposable arm64 Kubernetes 1.31+ cluster with UDS Core base and DNS matching the tenant gateway domain:

```bash
uds deploy educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.2.0.tar.zst \
  --set DOMAIN=uds.dev \
  --confirm
```

No Educates CLI binary is required on the host. The platform is independently deployable, but the current Python-to-UDS workshop still needs the reference implementation's UDS portal and dynamic session routing before this combined bundle is functionally complete.
