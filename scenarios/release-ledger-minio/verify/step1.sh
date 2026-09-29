#!/bin/bash
set -euo pipefail
cd "${SCENARIO_WORKSPACE:-/root/release-ledger-minio}"
grep -q 'class MinioContentStore' app.py
grep -q 'EVIDENCE_STORAGE' app.py
grep -q 'S3_ACCESS_KEY' app.py
grep -q 'S3_SECRET_KEY' app.py
echo 'Metadata/content boundary inspected'
