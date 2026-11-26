# Configuration

The Python-to-UDS workshop package exposes these Zarf package values:

| Key | Default | Description |
|---|---:|---|
| `portal.sessionsMaximum` | `1` | Maximum concurrent sessions across the portal. |
| `portal.anonymousMaximum` | `1` | Maximum concurrent anonymous sessions. |

Example standalone deployment override:

```bash
uds zarf package deploy zarf-package-educates-python-to-uds-arm64-0.1.0.tar.zst \
  --set-values portal.sessionsMaximum=2 \
  --set-values portal.anonymousMaximum=2 \
  --confirm
```

The package intentionally does not expose image, security-context, Docker privilege, ingress, or namespace-policy settings. Those remain pinned parts of the spike's security and runtime contract.
