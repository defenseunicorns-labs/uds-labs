#!/bin/bash
set -euo pipefail
export KUBECONFIG=/root/.kube/config
cd /root/release-ledger-postgres
test -n "$(find . -maxdepth 1 -name 'zarf-package-release-ledger-postgres-*.tar.zst' -print -quit)"
test -n "$(find bundle -maxdepth 1 -name 'uds-bundle-release-ledger-postgres-*.tar.zst' -print -quit)"
uds zarf tools kubectl get deployment release-ledger-postgres -n release-ledger-postgres >/dev/null
curl -skf https://release-ledger-postgres.uds.dev/health >/dev/null
echo 'PostgreSQL-backed application verified'
