# Step 6 – Deploy and observe ephemeral data

```bash
cd /root/release-ledger && uds run dev
```

This builds the Zarf package, creates the UDS bundle, and deploys Release Ledger.

## Watch the rollout

```bash
watch uds zarf tools kubectl get pods -n release-ledger
```

Wait for the pod to reach `Running` with `1/1` READY.

## Open the application

Click the **release-ledger** service chip in the browser panel. The application is available at:

```text
https://release-ledger.uds.dev
```

The app is intentionally unauthenticated in this baseline. A later scenario adds Authservice integration.

## Create a Release

```bash
curl -sk -X POST https://release-ledger.uds.dev/api/releases \
  -H 'Content-Type: application/json' \
  -d '{"application_name":"ledger-api","version":"1.0.0","environment":"staging"}'
curl -sk https://release-ledger.uds.dev/api/releases
```

## Replace the application pod

```bash
uds zarf tools kubectl delete pod -n release-ledger -l app=release-ledger
uds zarf tools kubectl wait -n release-ledger --for=condition=Ready pod -l app=release-ledger --timeout=120s
curl -sk https://release-ledger.uds.dev/api/releases
```

The Release is gone. Its metadata lived in the application process, not in durable storage. The next scenario moves Release metadata into operator-managed PostgreSQL.

## Verify

```bash
export KUBECONFIG=/root/.kube/config
uds zarf tools kubectl get namespace release-ledger >/dev/null && \
uds zarf tools kubectl get deployment release-ledger -n release-ledger >/dev/null && \
uds zarf tools kubectl get pods -n release-ledger --no-headers 2>/dev/null | \
  awk '$3=="Running"' | grep -q .
```
