# Step 1 – Separate metadata from content

```bash
cd /root/release-ledger-minio
grep -nE 'PostgresMetadataStore|MinioContentStore|EVIDENCE_STORAGE|S3_' app.py
```

Release records and evidence metadata use PostgreSQL. Evidence bytes use `FileContentStore` or `MinioContentStore`, selected by `EVIDENCE_STORAGE`.

Trace the required settings and identify which must be Secret references:

- ConfigMap: `EVIDENCE_STORAGE`, `S3_ENDPOINT`, and `S3_BUCKET`.
- Secret: `S3_ACCESS_KEY` and `S3_SECRET_KEY`.
- PostgreSQL retains its separate generated credentials.

## Verify

```bash
grep 'class MinioContentStore' app.py
grep 'EVIDENCE_STORAGE' app.py
grep 'S3_ACCESS_KEY' app.py
grep 'S3_SECRET_KEY' app.py
echo 'Metadata/content boundary inspected'
```
