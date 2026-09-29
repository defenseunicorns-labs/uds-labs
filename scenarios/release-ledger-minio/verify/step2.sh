#!/bin/bash
set -euo pipefail
cd /root/release-ledger-minio
grep -q 'ghcr.io/uds-packages/minio-operator' bundle/uds-bundle.yaml
grep -q 'copyPassword:' bundle/uds-bundle.yaml
grep -q 'bucketNames:' bundle/uds-bundle.yaml
grep -q 'evidence' bundle/uds-bundle.yaml
! grep -Eq 'secret_key:[[:space:]]*[^ ]' bundle/uds-bundle.yaml
echo 'Scoped MinIO app configured'
