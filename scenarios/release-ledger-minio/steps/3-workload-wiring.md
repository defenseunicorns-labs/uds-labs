# Step 3 – Bind endpoint and credentials

Inspect the chart wiring:

```bash
cd /root/release-ledger-minio
cat chart/templates/deployment.yaml
cat chart/templates/configmap.yaml
```

Use `configMapKeyRef` for endpoint, bucket, and mode. Use `secretKeyRef` for the copied `access_key` and `secret_key` keys. The PostgreSQL username and password remain separate Secret references too.

Render the workload and look for the absence of credential literals:

```bash
uds zarf tools helm template release-ledger chart/ > /tmp/release-ledger-minio.yaml
grep -A120 'kind: Deployment' /tmp/release-ledger-minio.yaml
```

## Verify

```bash
grep 'S3_ENDPOINT' chart/templates/configmap.yaml
grep 'S3_BUCKET' chart/templates/configmap.yaml
grep 'S3_ACCESS_KEY' chart/templates/deployment.yaml
grep 'S3_SECRET_KEY' chart/templates/deployment.yaml
grep 'secretKeyRef' chart/templates/deployment.yaml
! grep -R -Eq 'secret[_-]?key:[[:space:]]*[^" ]' chart bundle
```
