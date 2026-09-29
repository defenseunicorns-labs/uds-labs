# Step 4 – Configure reciprocal network access

The operator package creates PostgreSQL resources in its fixed `postgres` namespace. Cross-namespace PostgreSQL access requires both sides of the contract:

1. The application Package CR permits egress to the database Service on port 5432.
2. The PostgreSQL bundle override permits ingress from the application namespace and workload selector.

```bash
cd /root/release-ledger-postgres
sed -n '/network:/,$p' chart/templates/uds-package.yaml
sed -n '/ingress:/,$p' bundle/uds-bundle.yaml
```

Render the Package CR and inspect selectors, namespaces, and ports. Do not add `Anywhere` or a wildcard to make a failed connection pass.

## Verify

```bash
awk '/remoteNamespace: postgres/{found=1} found{print}' chart/templates/uds-package.yaml | grep -q 'port: 5432'
grep -q 'remoteNamespace: release-ledger-postgres' bundle/uds-bundle.yaml
grep -q 'remoteSelector:' bundle/uds-bundle.yaml
grep -q 'cluster-name: pg-cluster' chart/templates/uds-package.yaml
```
