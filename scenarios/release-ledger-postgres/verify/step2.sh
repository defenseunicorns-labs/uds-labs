#!/bin/bash
set -euo pipefail
cd /root/release-ledger-postgres
grep -q 'ghcr.io/uds-packages/postgres-operator' bundle/uds-bundle.yaml
grep -q 'users:' bundle/uds-bundle.yaml
grep -q 'databases:' bundle/uds-bundle.yaml
grep -q 'ingress:' bundle/uds-bundle.yaml
! grep -Eq 'password:[[:space:]]*[^" ]' bundle/uds-bundle.yaml
echo 'PostgreSQL dependency configured'
