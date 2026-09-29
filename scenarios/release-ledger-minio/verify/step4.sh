#!/bin/bash
set -euo pipefail
cd /root/release-ledger-minio
grep -q 'requests:' chart/values.yaml
grep -q 'limits:' chart/values.yaml
grep -q 'size: 5Gi' bundle/uds-bundle.yaml
grep -q 'volumesPerServer: 4' bundle/uds-bundle.yaml
echo 'Resource and storage profile configured'
