#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger

test -f chart/templates/deployment.yaml
test -f chart/templates/service.yaml
test -f chart/templates/uds-package.yaml
grep 'cpu: "4"' chart/templates/deployment.yaml
grep 'targetPort: 8081' chart/templates/service.yaml
grep 'release-ledger-api' chart/templates/uds-package.yaml
grep 'customer.example.com' chart/templates/uds-package.yaml
echo "Initial package inspected"
