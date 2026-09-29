# Step 5 – Verify object durability

Build and deploy from the final source:

```bash
cd /root/release-ledger-minio
uds run dev
```

Confirm the generated dependency resources and copied Secret:

```bash
uds zarf tools kubectl get pods -A
uds zarf tools kubectl get secret -n release-ledger-minio release-ledger-minio
```

Create a release and upload evidence:

```bash
RELEASE=$(curl -sk -X POST https://release-ledger-minio.uds.dev/api/releases \
  -H 'Content-Type: application/json' \
  -d '{"application_name":"ledger-api","version":"1.0.0","environment":"staging"}' | jq -r .id)
printf 'evidence survives replacement\n' >/tmp/evidence.txt
EVIDENCE=$(curl -sk -X POST "https://release-ledger-minio.uds.dev/api/releases/$RELEASE/evidence" -F file=@/tmp/evidence.txt)
echo "$EVIDENCE"
```

Download and checksum it, replace the application pod, then repeat the download. Finally delete the evidence and confirm the API no longer lists it:

```bash
uds zarf tools kubectl delete pod -n release-ledger-minio -l app=release-ledger-minio
uds zarf tools kubectl wait -n release-ledger-minio --for=condition=Ready pod -l app=release-ledger-minio --timeout=180s
curl -skf https://release-ledger-minio.uds.dev/health
```

Check pod restarts and application logs before declaring success. A successful deploy command alone is not health evidence.

## Verify

```bash
export KUBECONFIG=/root/.kube/config
uds zarf tools kubectl get deployment release-ledger-minio -n release-ledger-minio >/dev/null
curl -skf https://release-ledger-minio.uds.dev/health >/dev/null
echo 'MinIO-backed application verified'
```
