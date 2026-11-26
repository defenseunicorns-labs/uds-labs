#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

rm -rf "${HOME}/myapp"
cp -R "${HOME}/exercises/myapp" "${HOME}/myapp"

cat >>"${WORKSHOP_ENV}" <<EOF
export MYAPP_DIR="${HOME}/myapp"
EOF
