#!/bin/bash
set -euo pipefail
cd /root/release-ledger-postgres
grep -q 'remoteNamespace: postgres' chart/templates/uds-package.yaml
grep -q 'port: 5432' chart/templates/uds-package.yaml
grep -q 'remoteNamespace: release-ledger-postgres' bundle/uds-bundle.yaml
grep -q 'cluster-name: pg-cluster' chart/templates/uds-package.yaml
echo 'Reciprocal network contract configured'
