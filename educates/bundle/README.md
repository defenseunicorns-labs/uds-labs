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
uds deploy educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.3.1.tar.zst \
  --set DOMAIN=uds.dev \
  --confirm
```

No Educates CLI binary is required on the host. After deployment, run `uds run educates:verify-deployment`. The portal route should be present immediately; dashboard and editor routes appear after a learner allocates a session.
