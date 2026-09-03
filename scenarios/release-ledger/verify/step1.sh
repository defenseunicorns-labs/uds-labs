#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
cd /root/release-ledger
[[ -f app.py && -f requirements.txt && -f Dockerfile ]]
grep -q '/api/releases' app.py
grep -q '/health' app.py
