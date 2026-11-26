# Configuration

The Educates platform package exposes one Zarf package value:

| Key | Default | Description |
|---|---|---|
| `domain` | `educates.uds.dev` | Wildcard parent domain used for portals, sessions, and application ingresses. |

Deploy the standalone package with a non-default value:

```bash
uds zarf package deploy zarf-package-educates-arm64-3.8.0-uds.0.tar.zst \
  --set-values domain=educates.example.com \
  --confirm
```

The spike bundle maps its deploy-time `DOMAIN` variable to this chart value:

```bash
uds deploy uds-bundle-educates-uds-labs-spike-arm64-0.1.0.tar.zst \
  --set DOMAIN=educates.example.com \
  --confirm
```

Wildcard DNS for the selected domain must resolve to the external address of the `envoy` LoadBalancer Service in the `projectcontour` namespace.
