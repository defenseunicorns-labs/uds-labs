#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

EDUCATES_VERSION=3.8.0
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp)"
trap 'rm -f "${tmp}"' EXIT

docker run --rm \
  --tmpfs /tmp:rw,exec \
  --volume "${root}/config:/config:ro" \
  "ghcr.io/educates/educates-cli:${EDUCATES_VERSION}" \
  admin platform deploy \
  --config /config/installer.yaml \
  --version "${EDUCATES_VERSION}" \
  --package-repository ghcr.io/educates \
  --dry-run \
  --skip-image-resolution \
  >"${tmp}"

sed 's/###ZARF_VAR_DOMAIN###/__EDUCATES_DOMAIN__/g' "${tmp}" \
  >"${root}/chart/files/educates.yaml"
trap - EXIT

printf 'Rendered Educates %s manifests to %s\n' \
  "${EDUCATES_VERSION}" "${root}/chart/files/educates.yaml"
