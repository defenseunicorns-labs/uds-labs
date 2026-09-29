#!/bin/bash
set -euo pipefail
export KUBECONFIG=/root/.kube/config
cd /root/release-ledger-minio
test -n "$(find . -maxdepth 1 -name 'zarf-package-release-ledger-minio-*.tar.zst' -print -quit)"
test -n "$(find bundle -maxdepth 1 -name 'uds-bundle-release-ledger-minio-*.tar.zst' -print -quit)"
uds zarf tools kubectl get deployment release-ledger-minio -n release-ledger-minio >/dev/null
curl -skf https://release-ledger-minio.uds.dev/health >/dev/null
echo 'MinIO-backed application verified'
