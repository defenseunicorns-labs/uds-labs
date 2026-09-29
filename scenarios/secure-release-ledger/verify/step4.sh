#!/bin/bash
set -euo pipefail
export KUBECONFIG=/root/.kube/config
cd /root/secure-release-ledger
uds zarf tools kubectl get package secure-release-ledger -n secure-release-ledger >/dev/null
uds zarf tools kubectl get deployment secure-release-ledger -n secure-release-ledger >/dev/null
echo 'Protected package deployed'
