#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

: "${IMAGE_REPOSITORY:?Set IMAGE_REPOSITORY, for example ghcr.io/your-org}"
WORKSHOP_VERSION="${WORKSHOP_VERSION:-dev}"

command -v educates >/dev/null || {
  echo "educates CLI 3.8.0 is required" >&2
  exit 1
}

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${root}/workshop-${WORKSHOP_VERSION}.yaml"

educates publish-workshop "${root}" \
  --image-repository "${IMAGE_REPOSITORY}" \
  --workshop-version "${WORKSHOP_VERSION}" \
  --export-workshop "${output}"

printf 'Published workshop content and wrote %s\n' "${output}"
printf 'Deploy with: educates deploy-workshop -f %q\n' "${output}"
