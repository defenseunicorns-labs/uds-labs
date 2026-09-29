#!/bin/bash
set -euo pipefail
cd /root/secure-release-ledger
grep -A3 'enableAuthserviceSelector:' chart/templates/uds-package.yaml | grep -q 'app: secure-release-ledger'
grep -A4 'selector:' chart/templates/service.yaml | grep -q 'app: secure-release-ledger'
grep -q 'host: secure-release-ledger' chart/templates/uds-package.yaml
echo 'Selector and route contract aligned'
