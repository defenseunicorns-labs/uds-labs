# Step 3 – Repair Service and exposure

Trace the request from the application to the UDS tenant gateway:

```text
application listen port -> containerPort -> Service targetPort -> Service port -> UDS exposure
```

The Service should keep the package-owned name `release-ledger`, use the configured Service port,
and target the named `http` container port. The UDS Package CR must reference that same Service.

Repair these values:

- `Service.spec.ports[].port` should use `.Values.service.port`.
- `Service.spec.ports[].targetPort` should use `.Values.service.targetPort`.
- `network.expose[].service` should be `release-ledger`.
- `network.expose[].host` should be the stable host prefix `release-ledger`, not a full customer
  domain.
- The exposure should use the Service port. The current uptime schema checks paths and does not take a port.

The UDS lab resolves the resulting application route at `release-ledger.uds.dev`. The domain suffix
belongs to the deployment environment; the package owns the stable application host prefix.

Open the built-in Web IDE with the **IDE** button. Edit these two files manually; do not run a
text-replacement script in the terminal.

In `chart/templates/service.yaml`, update the Service ports to use the chart values:

```yaml
ports:
  - name: http
    port: {{ .Values.service.port }}
    targetPort: {{ .Values.service.targetPort }}
```

In `chart/templates/uds-package.yaml`, update the exposure to reference the package-owned Service,
the stable host prefix, and the configured Service port:

```yaml
expose:
  - description: Release Ledger application
    service: release-ledger
    selector:
      app: release-ledger
    gateway: tenant
    host: release-ledger
    port: {{ .Values.service.port }}
```

Save both files in the IDE. The examples above are display-only; make the edits in the existing
files rather than replacing them wholesale.

Render the chart:

```bash
uds zarf tools helm template release-ledger chart/ > /tmp/release-ledger-rendered.yaml
```

Inspect the rendered Service:

```bash
grep -A8 -B2 'kind: Service' /tmp/release-ledger-rendered.yaml
```

Inspect the rendered Package CR:

```bash
grep -A18 -B2 'kind: Package' /tmp/release-ledger-rendered.yaml
```

## Verify

```bash
grep 'port: {{ .Values.service.port }}' chart/templates/service.yaml
```

```bash
grep 'targetPort: {{ .Values.service.targetPort }}' chart/templates/service.yaml
```

```bash
grep '^        service: release-ledger$' chart/templates/uds-package.yaml
```

```bash
grep '^        host: release-ledger$' chart/templates/uds-package.yaml
```

```bash
grep '^        port: {{ .Values.service.port }}$' chart/templates/uds-package.yaml
```

The Service and exposure contract is repaired when each check succeeds.
