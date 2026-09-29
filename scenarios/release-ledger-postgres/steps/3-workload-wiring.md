# Step 3 – Bind ConfigMap and Secret values

The Deployment should consume non-secret settings from `release-ledger-config` and credentials from the operator-created Secret. This checkpoint is a configuration review: the starter chart already contains the intended wiring, so preserve the Helm value references rather than replacing them with literal credentials. If your workspace does not contain these references, add the mappings before continuing.

```bash
cd /root/release-ledger-postgres
sed -n '1,260p' chart/templates/deployment.yaml
sed -n '1,160p' chart/templates/configmap.yaml
```

Use `valueFrom.configMapKeyRef` for `STORAGE_MODE`, `DATABASE_HOST`, `DATABASE_PORT`, and `DATABASE_NAME`. Use `valueFrom.secretKeyRef` for `DATABASE_USER` and `DATABASE_PASSWORD`, with keys `username` and `password` from the operator Secret.

The consuming namespace is `release-ledger-postgres`; do not look for this Secret in the operator namespace. Render and inspect the workload:

```bash
uds zarf tools helm template release-ledger chart/ > /tmp/release-ledger-postgres.yaml
grep -A90 'kind: Deployment' /tmp/release-ledger-postgres.yaml
```

## Verify

```bash
grep -q 'kind: ConfigMap' chart/templates/configmap.yaml
grep -q 'configMapKeyRef' chart/templates/deployment.yaml
grep -q 'secretKeyRef' chart/templates/deployment.yaml
grep -q 'key: "username"' chart/templates/deployment.yaml
grep -q 'key: "password"' chart/templates/deployment.yaml
! grep -R -Eq 'password:[[:space:]]*[^" ]' chart bundle
```
