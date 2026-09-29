#!/bin/bash
set -euo pipefail
cd /root/secure-release-ledger
grep -q 'sso:' chart/templates/uds-package.yaml
grep -q 'clientId: uds-secure-release-ledger' chart/templates/uds-package.yaml
grep -q '/login' chart/templates/uds-package.yaml
grep -q 'enableAuthserviceSelector:' chart/templates/uds-package.yaml
echo 'Authservice client configured'
