#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

: "${IMAGE_REPOSITORY:?Set IMAGE_REPOSITORY, for example ghcr.io/your-org}"
WORKSHOP_VERSION="${WORKSHOP_VERSION:-dev}"
TARGET_PLATFORMS="${TARGET_PLATFORMS:-linux/arm64}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
image="${IMAGE_REPOSITORY}/educates-python-to-uds-image:${WORKSHOP_VERSION}"

if [[ "${TARGET_PLATFORMS}" == *,* ]]; then
  docker buildx build \
    --platform "${TARGET_PLATFORMS}" \
    --tag "${image}" \
    --push \
    "${root}"
else
  # A daemon build followed by docker push works with Educates' local
  # localhost:5001 registry. A container-backed buildx builder cannot reach
  # that registry through its own loopback interface.
  docker build \
    --platform "${TARGET_PLATFORMS}" \
    --tag "${image}" \
    "${root}"
  docker push "${image}"
fi

printf 'Published %s for %s\n' "${image}" "${TARGET_PLATFORMS}"
