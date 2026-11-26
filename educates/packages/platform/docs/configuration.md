# Configuration

Educates 4.0.0-alpha.10 requires Kubernetes 1.31 or newer, UDS Core base with ambient mesh and a tenant gateway, a Zarf-initialized registry, and a default StorageClass.

The package exposes one Zarf package value:

| Key | Default | Description |
|---|---|---|
| `domain` | `uds.dev` | Domain shared by Educates-generated URLs and the UDS tenant gateway. |

Deploy the standalone bundle with a non-default value:

```bash
uds deploy bundle/uds-bundle-educates-platform-arm64-4.0.0-alpha.10-uds.0.tar.zst \
  --set DOMAIN=example.com \
  --confirm
```

For direct Zarf package deployment, use `--set-values domain=example.com`.

The upstream chart's Ingress objects remain inactive through the controllerless `educates-unused` IngressClass. Each workshop package is responsible for UDS `Package` exposures for its portal and dynamically created sessions.

Only session-manager and secrets-manager are enabled. Lookup service, remote access, node CA injection, image pre-pull, upstream policy engines, and optional registry/assets features are disabled. UDS owns policy and mesh enrollment.

The chart uses `skipWait` because statusless `SecretInjector` resources cannot satisfy generic kstatus waiting. Zarf instead health-checks the session-manager and secrets-manager Deployments.
