#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger
[[ -f chart/templates/uds-package.yaml ]]
grep -q 'kind: Package' chart/templates/uds-package.yaml
grep -q 'network:' chart/templates/uds-package.yaml
grep -q 'expose:' chart/templates/uds-package.yaml
grep -q 'remoteGenerated' chart/templates/uds-package.yaml
