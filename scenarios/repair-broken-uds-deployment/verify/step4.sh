#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger

INGRESS_RULE=$(awk '/direction: Ingress/{in_rule=1} /direction: Egress/{in_rule=0} in_rule{print}' \
  chart/templates/uds-package.yaml)
printf '%s\n' "$INGRESS_RULE" | grep 'app: release-ledger'
printf '%s\n' "$INGRESS_RULE" | grep 'remoteGenerated: IntraNamespace'
printf '%s\n' "$INGRESS_RULE" | grep 'port: {{ .Values.service.port }}'
echo "Network policy contract repaired"
