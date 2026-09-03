#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger
grep -q 'UDSBundle' bundle/uds-bundle.yaml
grep -q 'name: release-ledger' bundle/uds-bundle.yaml
grep -q 'uds-common' tasks.yaml
grep -q 'name: dev' tasks.yaml
