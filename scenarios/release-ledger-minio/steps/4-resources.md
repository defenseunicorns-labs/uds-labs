# Step 4 – Apply production-shaped constraints

The lab profile is intentionally small. MinIO standalone mode still requires one server with four volumes; reducing it to one volume causes the tenant health check to reset connections. Read the values and dependency overrides:

```bash
cd /root/release-ledger-minio
cat chart/values.yaml
awk '/minio-tenant:/{show=1} show{print}' bundle/uds-bundle.yaml
```

The deployed profile should use one application replica, explicit CPU/memory requests and limits, and a small storage pool. For a production comparison, render a separate values overlay rather than silently deploying large resources in the lab.

```bash
uds zarf tools helm template release-ledger chart/ --set resources.requests.cpu=250m --set resources.limits.cpu=1
```

The application package remains environment-agnostic: operators can override resources, storage size, tenant count, and endpoints at the bundle/environment boundary.

## Verify

```bash
grep 'requests:' chart/values.yaml
grep 'limits:' chart/values.yaml
grep 'size: 5Gi' bundle/uds-bundle.yaml
grep 'volumesPerServer: 4' bundle/uds-bundle.yaml
```
