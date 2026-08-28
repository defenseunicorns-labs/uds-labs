#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

# Pass when the app source files exist in the working directory.
[[ -f /root/myapp/app.py ]] || exit 1
[[ -f /root/myapp/requirements.txt ]] || exit 1
[[ -f /root/myapp/Dockerfile ]] || exit 1
exit 0
