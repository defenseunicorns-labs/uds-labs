#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger
[[ -f chart/Chart.yaml && -f chart/templates/deployment.yaml && -f chart/templates/service.yaml ]]
grep -q 'kind: Deployment' chart/templates/deployment.yaml
grep -q 'app: release-ledger' chart/templates/deployment.yaml
grep -q 'containerPort: 8080' chart/templates/deployment.yaml
grep -q 'targetPort: 8080' chart/templates/service.yaml
