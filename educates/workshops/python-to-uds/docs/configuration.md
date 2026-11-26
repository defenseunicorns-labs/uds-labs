# Configuration

Requires Kubernetes 1.31 or newer, UDS Core base with ambient mesh and the tenant gateway, Zarf initialization, a default StorageClass, and the standalone Educates platform package at `4.0.0-alpha.10-uds.0`.

The Python-to-UDS workshop package exposes these Zarf package values:

| Key | Default | Description |
|---|---:|---|
| `portal.sessionsMaximum` | `1` | Maximum concurrent sessions across the portal. |
| `portal.anonymousMaximum` | `1` | Maximum concurrent anonymous sessions. |

Example standalone deployment override:

```bash
uds zarf package deploy zarf-package-educates-python-to-uds-arm64-0.2.0-uds.2.tar.zst \
  --set-values portal.sessionsMaximum=2 \
  --set-values portal.anonymousMaximum=2 \
  --confirm
```

The portal is exposed through the tenant gateway at `https://uds-labs-ui.<tenant-domain>`. The portal name, host, image, security context, Docker privilege, ports, and namespace-policy settings are intentionally pinned rather than exposed as package configuration.

Educates creates the `uds-labs-ui` namespace. The package waits for it before installing the portal UDS `Package`. The Workshop definition uses supported `environment.objects` and `session.objects` to create:

- one Package in each shared workshop-environment namespace;
- one protected routing namespace per session;
- one internal Istio ServiceEntry selecting the session backend;
- one Package per routing namespace exposing dashboard and editor hosts.

New Workshop generations replace environments for future sessions while active sessions drain. Docker-in-Docker is privileged; deploy only to a disposable, dedicated cluster.

Upstream Workshop reference: https://docs.educates.dev/en/latest/custom-resources/workshop-definition.html
