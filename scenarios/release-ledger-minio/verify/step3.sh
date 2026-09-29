#!/bin/bash
set -euo pipefail
cd /root/release-ledger-minio
grep -q 'S3_ENDPOINT' chart/templates/configmap.yaml
grep -q 'S3_BUCKET' chart/templates/configmap.yaml
grep -q 'S3_ACCESS_KEY' chart/templates/deployment.yaml
grep -q 'S3_SECRET_KEY' chart/templates/deployment.yaml
grep -q 'secretKeyRef' chart/templates/deployment.yaml
! grep -R -Eq 'secret[_-]?key:[[:space:]]*[^" ]' chart bundle
echo 'Object-storage wiring configured'
