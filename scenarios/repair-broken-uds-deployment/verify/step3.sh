#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger

grep 'port: {{ .Values.service.port }}' chart/templates/service.yaml
grep 'targetPort: {{ .Values.service.targetPort }}' chart/templates/service.yaml
grep '^        service: release-ledger$' chart/templates/uds-package.yaml
grep '^        host: release-ledger$' chart/templates/uds-package.yaml
grep '^        port: {{ .Values.service.port }}$' chart/templates/uds-package.yaml
echo "Service and exposure contract repaired"
