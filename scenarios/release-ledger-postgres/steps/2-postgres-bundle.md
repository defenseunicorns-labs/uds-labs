# Step 2 – Compose PostgreSQL in the bundle

The PostgreSQL operator belongs in `bundle/uds-bundle.yaml`, not in the application Zarf package. Use the pinned public package already present in the fixture and inspect its override contract.

```bash
cd /root/release-ledger-postgres
sed -n '1,240p' bundle/uds-bundle.yaml
```

The package intentionally has resources in both `postgres-operator` and `postgres` namespaces. Do not add a bundle-level `namespace` override: Zarf only permits that override for packages with one unique namespace. The database Service and Package CR therefore live in the operator package's fixed `postgres` namespace.

The important values are:

- `users` uses `<consuming-namespace>.<application-role>`.
- `databases` maps an explicit database name to that role.
- `ingress` selects the application from the database side.
- The cluster is intentionally small for a lab; production values are a separate comparison, not a reason to oversize this VM.

Confirm the expected generated Secret name from the operator documentation and then confirm the actual name after deployment. Remember that role underscores are normalized to dashes in the Secret name.

Keep the application package last so the dependency is composed first. Do not add a password to this file.

## Verify

```bash
grep -q 'ghcr.io/uds-packages/postgres-operator' bundle/uds-bundle.yaml
grep -q '1.15.1-uds.6-upstream' bundle/uds-bundle.yaml
grep -q 'users:' bundle/uds-bundle.yaml
grep -q 'databases:' bundle/uds-bundle.yaml
grep -q 'ingress:' bundle/uds-bundle.yaml
! grep -Eq 'password:[[:space:]]*[^" ]' bundle/uds-bundle.yaml
```
