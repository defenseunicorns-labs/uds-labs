#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

# Pass when the Package CR phase is Ready.
export HOME=/root
export KUBECONFIG=/root/.kube/config
PHASE=$(uds zarf tools kubectl get package reference-package -n reference-package \
  -o jsonpath='{.status.phase}' 2>/dev/null)
[[ "$PHASE" == "Ready" ]]
