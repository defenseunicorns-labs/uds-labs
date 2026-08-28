#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

# Pass when zarf.yaml exists with the correct kind and image reference.
[[ -f /root/myapp/zarf.yaml ]] || exit 1
grep -q "ZarfPackageConfig" /root/myapp/zarf.yaml || exit 1
grep -q "myapp" /root/myapp/zarf.yaml || exit 1
grep -q "myapp:dev" /root/myapp/zarf.yaml || exit 1
exit 0
