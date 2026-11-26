#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

EDUCATES_VERSION=3.8.0
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

mkdir -p "${tmp}/source"
cp -R "${root}/." "${tmp}/source/"
chmod -R a+rwX "${tmp}"

validator_image="uds-labs-educates-validator:${EDUCATES_VERSION}"
docker build --quiet \
  --file "${root}/scripts/Dockerfile.validate" \
  --tag "${validator_image}" \
  "${root}/scripts" \
  >/dev/null

if ! docker run --rm \
  --tmpfs /tmp:rw,exec \
  --volume "${tmp}:/validation" \
  --workdir /validation/source \
  "${validator_image}" \
  render-workshop . \
  --output-file /validation/workshop.zip \
  >"${tmp}/render.log" 2>&1; then
  tr -d '\000' <"${tmp}/render.log" >&2
  exit 1
fi

unzip -t "${tmp}/workshop.zip" >/dev/null
helm lint "${root}/chart"
helm template educates-python-to-uds "${root}/chart" --namespace educates \
  >"${tmp}/workshop-manifests.yaml"

grep -q 'kind: Workshop' "${tmp}/workshop-manifests.yaml"
grep -q 'kind: TrainingPortal' "${tmp}/workshop-manifests.yaml"
grep -q 'arm64)' "${root}/Dockerfile"
grep -q 'amd64)' "${root}/Dockerfile"

while IFS= read -r script; do
  [ -x "${script}" ] || {
    echo "setup and examiner scripts must be executable: ${script}" >&2
    exit 1
  }
done < <(find "${root}/workshop/setup.d" "${root}/workshop/examiner/tests" -type f -print)

if command -v shellcheck >/dev/null; then
  find "${root}" -type f -name '*.sh' -print0 | xargs -0 shellcheck
fi

if command -v uds >/dev/null; then
  uds zarf dev lint "${root}"
fi

printf 'Educates %s workshop and package validation passed via the CLI container.\n' \
  "${EDUCATES_VERSION}"
