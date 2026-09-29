# Step 5 – Verify data survives replacement

Build the package and bundle from the declarative source, then deploy:

```bash
cd /root/release-ledger-postgres
uds run dev
```

Check the authoritative platform state and find the generated credential Secret:

```bash
uds zarf tools kubectl get package release-ledger-postgres -n release-ledger-postgres
uds zarf tools kubectl get pods -n release-ledger-postgres
uds zarf tools kubectl get secrets -n release-ledger-postgres
uds zarf tools kubectl get pods -n postgres
```

Create a release through ingress and save its ID:

```bash
RELEASE=$(curl -sk -X POST https://release-ledger-postgres.uds.dev/api/releases \
  -H 'Content-Type: application/json' \
  -d '{"application_name":"ledger-api","version":"1.0.0","environment":"staging"}')
echo "$RELEASE"
```

Replace only the application pod and read the list again:

```bash
uds zarf tools kubectl delete pod -n release-ledger-postgres -l app=release-ledger-postgres
uds zarf tools kubectl wait -n release-ledger-postgres --for=condition=Ready pod -l app=release-ledger-postgres --timeout=180s
curl -skf https://release-ledger-postgres.uds.dev/api/releases
```

The release remains because its metadata is in PostgreSQL, not the process heap. Verify logs and restarts before declaring success.

## Verify

```bash
export KUBECONFIG=/root/.kube/config
uds zarf tools kubectl get deployment release-ledger-postgres -n release-ledger-postgres >/dev/null
curl -skf https://release-ledger-postgres.uds.dev/health >/dev/null
```
