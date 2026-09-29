# Step 4 – Repair network policy

A Package CR can render successfully while selecting the wrong workload. The ingress rule in this
package uses a selector that does not match the Deployment and Service labels.

Repair the ingress rule so that it:

- Selects `app: release-ledger`.
- Uses `remoteGenerated: IntraNamespace`.
- Allows only the configured Service port.

Keep the egress rule scoped to the namespace. Do not replace a selector with `Anywhere` just to make
a request succeed.

Open `chart/templates/uds-package.yaml` in the built-in Web IDE and manually update only the
Ingress rule. Do not run a text-replacement script in the terminal.

The completed Ingress rule should look like this display-only example:

```yaml
allow:
  - direction: Ingress
    selector:
      app: release-ledger
    remoteGenerated: IntraNamespace
    port: {{ .Values.service.port }}
    description: Allow application ingress
```

Keep the egress rule unchanged. Save the file in the IDE. The selector, direction, remote target,
and port in the saved file are the contract UDS Core will use.

Render the chart:

```bash
uds zarf tools helm template release-ledger chart/ > /tmp/release-ledger-rendered.yaml
```

Inspect the complete rendered Package CR:

```bash
sed -n '/kind: Package/,$p' /tmp/release-ledger-rendered.yaml
```

Then render again after the edit. Treat the rendered selector, direction, remote target, and port as
the contract that UDS Core will use to generate policy resources.

## Verify

```bash
awk '/direction: Ingress/{in_rule=1} /direction: Egress/{in_rule=0} in_rule{print}' chart/templates/uds-package.yaml | grep 'app: release-ledger'
```

```bash
awk '/direction: Ingress/{in_rule=1} /direction: Egress/{in_rule=0} in_rule{print}' chart/templates/uds-package.yaml | grep 'remoteGenerated: IntraNamespace'
```

```bash
awk '/direction: Ingress/{in_rule=1} /direction: Egress/{in_rule=0} in_rule{print}' chart/templates/uds-package.yaml | grep 'port: {{ .Values.service.port }}'
```

The network policy contract is repaired when each check succeeds.
