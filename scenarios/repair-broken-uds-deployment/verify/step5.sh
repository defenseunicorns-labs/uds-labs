#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
export KUBECONFIG=/root/.kube/config
cd /root/release-ledger
KUBECTL="uds zarf tools kubectl"

test -n "$(find . -maxdepth 1 -type f -name 'zarf-package-release-ledger-repair-*.tar.zst' -print -quit)"
test -n "$(find bundle -maxdepth 1 -type f -name 'uds-bundle-release-ledger-repair-*.tar.zst' -print -quit)"
$KUBECTL get deployment release-ledger -n release-ledger >/dev/null
AVAILABLE=$($KUBECTL get deployment release-ledger -n release-ledger -o jsonpath='{.status.availableReplicas}')
[[ "${AVAILABLE:-0}" -ge 1 ]]

POD=$($KUBECTL get pods -n release-ledger -l app=release-ledger \
  -o jsonpath='{.items[0].metadata.name}')
RESTARTS=$($KUBECTL get pod "$POD" -n release-ledger \
  -o jsonpath='{.status.containerStatuses[0].restartCount}')
[[ "${RESTARTS:-1}" == "0" ]]

curl -skf https://release-ledger.uds.dev/health >/dev/null
curl -skf https://release-ledger.uds.dev/api/releases >/dev/null
echo "Repaired package deployed and verified"
