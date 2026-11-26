#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-ghcr.io/defenseunicorns-labs}"
WORKSHOP_VERSION="${WORKSHOP_VERSION:-0.1.0}"
TARGET_PLATFORMS="${TARGET_PLATFORMS:-linux/arm64}"
PUSH_IMAGE="${PUSH_IMAGE:-0}"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
image="${IMAGE_REPOSITORY}/educates-python-to-uds-image:${WORKSHOP_VERSION}"

if [ "${PUSH_IMAGE}" = "1" ] && [[ "${TARGET_PLATFORMS}" == *,* ]]; then
  docker buildx build \
    --platform "${TARGET_PLATFORMS}" \
    --tag "${image}" \
    --push \
    "${root}"
else
  docker build \
    --platform "${TARGET_PLATFORMS}" \
    --tag "${image}" \
    "${root}"
  if [ "${PUSH_IMAGE}" = "1" ]; then
    docker push "${image}"
  fi
fi

printf 'Built %s for %s (push=%s)\n' \
  "${image}" "${TARGET_PLATFORMS}" "${PUSH_IMAGE}"
