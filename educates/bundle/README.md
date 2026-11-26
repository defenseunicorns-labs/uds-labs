# Educates UDS Labs spike bundle

The bundle installs two separate packages in order:

1. `educates` — the Educates 3.8.0 platform, Contour, and Kyverno.
2. `educates-python-to-uds` — the arm64 workshop image and its Educates resources.

Create both Zarf packages and the bundle with:

```bash
uds run educates:create-bundle
```

Deploy to an arm64 cluster with wildcard DNS pointing at the Contour LoadBalancer:

```bash
uds deploy educates/bundle/uds-bundle-educates-uds-labs-spike-arm64-0.1.0.tar.zst \
  --set DOMAIN=educates.example.com \
  --confirm
```

No Educates CLI binary is required on the host.
