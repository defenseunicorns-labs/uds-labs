#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

EDUCATES_VERSION=3.8.0
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
cleanup() {
  rm -rf "${tmp}"
  rm -f "${root}/workshop/.hugo_build.lock"
}
trap cleanup EXIT

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *)
    echo "unsupported validation host OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *)
    echo "unsupported validation host architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

case "${os}/${arch}" in
  linux/amd64) sha256=4764303d0726380c1a4c1a03d38cd6b87685598757f146739317d413f2d8ed08 ;;
  linux/arm64) sha256=0323a2b2d37986ff83280a507dc81bbc6f797e6875e785557e3c32b94f5ff8ed ;;
  darwin/amd64) sha256=dd343c48038e629add095558ecf27d14caab4663b390e96d0ddc9ff0b61a9d24 ;;
  darwin/arm64) sha256=c688add711c544f9c366a2981fab6d0aad8a3d7a3d41cff5185c4e4c74bc5235 ;;
esac

educates_bin="${tmp}/educates"
curl --fail --location --silent --show-error \
  --output "${educates_bin}" \
  "https://github.com/educates/educates-training-platform/releases/download/${EDUCATES_VERSION}/educates-${os}-${arch}"
if command -v sha256sum >/dev/null; then
  actual_sha256="$(sha256sum "${educates_bin}" | awk '{print $1}')"
else
  actual_sha256="$(shasum -a 256 "${educates_bin}" | awk '{print $1}')"
fi
[ "${actual_sha256}" = "${sha256}" ] || {
  echo "Educates CLI checksum mismatch" >&2
  exit 1
}
chmod 0755 "${educates_bin}"

if ! "${educates_bin}" render-workshop "${root}" \
  --image-repository registry.example.test/uds-labs \
  --workshop-version validation \
  --output-file "${tmp}/workshop.zip" \
  >"${tmp}/render.log" 2>&1; then
  tr -d '\000' <"${tmp}/render.log" >&2
  exit 1
fi

unzip -t "${tmp}/workshop.zip" >/dev/null

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

printf 'Educates %s workshop render passed (%s validation host).\n' \
  "${EDUCATES_VERSION}" "${arch}"
