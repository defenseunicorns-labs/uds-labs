#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger

deployment=chart/templates/deployment.yaml

check() {
  local description="$1"
  shift
  if ! "$@"; then
    echo "Step 2 check failed: ${description}" >&2
    exit 1
  fi
}

check 'replicas use .Values.replicaCount' \
  grep -E 'replicas:.*\.Values\.replicaCount' "$deployment"
check 'pod fsGroup is 1000' \
  grep -E 'fsGroup:[[:space:]]*1000' "$deployment"
check 'EVIDENCE_DIR uses .Values.evidenceDir' \
  grep -E 'value:.*\.Values\.evidenceDir' "$deployment"
check 'containerPort uses .Values.containerPort' \
  grep -E 'containerPort:.*\.Values\.containerPort' "$deployment"
check 'resources use .Values.resources' \
  grep -E 'toYaml[[:space:]]+\.Values\.resources' "$deployment"
check 'evidence mountPath uses .Values.evidenceDir' \
  grep -E 'mountPath:.*\.Values\.evidenceDir' "$deployment"
check 'evidence volume uses emptyDir' \
  grep 'emptyDir: {}' "$deployment"

probe_count=$(grep -Ec 'port:[[:space:]]*http' "$deployment" || true)
if [[ "$probe_count" -lt 2 ]]; then
  echo "Step 2 check failed: both probes must use the named http port" >&2
  exit 1
fi

echo "Workload contract repaired"
