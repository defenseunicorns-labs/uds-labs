#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger
grep -q 'ZarfPackageConfig' zarf.yaml
grep -q 'release-ledger:dev' zarf.yaml
