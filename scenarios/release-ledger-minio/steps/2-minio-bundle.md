# Step 2 – Provision a scoped MinIO app

```bash
cd /root/release-ledger-minio
cat bundle/uds-bundle.yaml
```

The MinIO package's `apps` override provisions an application-scoped bucket and user. `copyPassword` copies only that user's credentials into the application namespace with the chosen Secret and key names.

The built-in tenant uses a small lab pool. Keep the MinIO package separate from the application package and place dependencies before the application package in the bundle.

Use the package documentation to confirm the tenant service name and the copied Secret contents after deployment. Do not use the root credential or write a literal credential into the bundle.

## Verify

```bash
grep 'ghcr.io/uds-packages/minio-operator' bundle/uds-bundle.yaml
```
```bash
grep '7.1.1-uds.25-upstream' bundle/uds-bundle.yaml
```
```bash
grep -A4 'copyPassword:' bundle/uds-bundle.yaml
```
```bash
grep -A1 'bucketNames:' bundle/uds-bundle.yaml
```

```bash
grep 'secret_key' bundle/uds-bundle.yaml
```
