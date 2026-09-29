#!/bin/bash
set -euo pipefail
export KUBECONFIG=/root/.kube/config
cd /root/secure-release-ledger
curl -skI https://secure-release-ledger.uds.dev/ | grep -Eq 'HTTP/.*30[123]'
uds zarf tools kubectl get package secure-release-ledger -n secure-release-ledger -o jsonpath='{.status.phase}' | grep -q Ready
echo 'Authservice protection verified'
