#!/bin/bash
set -euo pipefail
cd /root/secure-release-ledger
grep -q 'def create_app' app.py
grep -q 'kind: Service' chart/templates/service.yaml
grep -q 'app: secure-release-ledger' chart/templates/deployment.yaml
echo 'Auth boundary identified'
