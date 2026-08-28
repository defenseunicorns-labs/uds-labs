#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

# Pass when the repo is cloned and key directories exist.
export HOME=/root
[[ -f /root/reference-package/zarf.yaml ]] || exit 1
[[ -d /root/reference-package/chart ]] || exit 1
[[ -d /root/reference-package/bundle ]] || exit 1
exit 0
