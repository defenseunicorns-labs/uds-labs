#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
export HOME=/root
export KUBECONFIG=/root/.kube/config
KUBECTL="uds zarf tools kubectl"
$KUBECTL get namespace release-ledger >/dev/null
$KUBECTL get deployment release-ledger -n release-ledger >/dev/null
RUNNING=$($KUBECTL get pods -n release-ledger --no-headers 2>/dev/null | awk '$3=="Running"' | wc -l)
[[ "$RUNNING" -ge 1 ]]
