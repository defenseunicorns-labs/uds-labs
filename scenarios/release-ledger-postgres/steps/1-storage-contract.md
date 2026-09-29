# Step 1 – Trace the storage contract

```bash
cd /root/release-ledger-postgres
sed -n '1,260p' app.py
```

Find `PostgresMetadataStore` and `create_metadata_store`. The application expects separate `DATABASE_HOST`, `DATABASE_NAME`, `DATABASE_USER`, and `DATABASE_PASSWORD` settings. It does not expect a `DATABASE_URL` and it does not need the `postgres` superuser.

Classify the settings:

- `STORAGE_MODE`, host, port, and database name are non-secret configuration.
- The generated username and password are secrets.
- The application package may declare Secret names and keys, but never credential values.

Inspect the starting chart and render it before changing anything:

```bash
sed -n '1,220p' chart/values.yaml
uds zarf tools helm template release-ledger chart/
```

## Verify

```bash
test -f app.py
grep -q 'DATABASE_USER' app.py
grep -q 'DATABASE_PASSWORD' app.py
grep -q 'STORAGE_MODE' app.py
```
