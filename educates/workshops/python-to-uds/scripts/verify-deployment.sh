#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

set -euo pipefail

kubectl=(uds zarf tools kubectl)

"${kubectl[@]}" rollout status deployment/session-manager \
  -n educates-platform --timeout=180s
"${kubectl[@]}" rollout status deployment/secrets-manager \
  -n educates-platform --timeout=180s
"${kubectl[@]}" rollout status deployment/training-portal \
  -n uds-labs-ui --timeout=300s

platform_phase=$("${kubectl[@]}" get package educates-platform \
  -n educates-platform -o jsonpath='{.status.phase}')
portal_phase=$("${kubectl[@]}" get package educates-portal \
  -n uds-labs-ui -o jsonpath='{.status.phase}')
portal_runtime=$("${kubectl[@]}" get trainingportal uds-labs \
  -o jsonpath='{.status.educates.phase}')

[[ ${platform_phase} == Ready ]]
[[ ${portal_phase} == Ready ]]
[[ ${portal_runtime} == Running ]]

portal_hosts=$("${kubectl[@]}" get virtualservice -n uds-labs-ui \
  -o jsonpath='{range .items[*].spec.hosts[*]}{.}{"\n"}{end}')
grep -q '^uds-labs-ui\.' <<<"${portal_hosts}"

printf 'Platform Package: %s\n' "${platform_phase}"
printf 'Portal Package: %s\n' "${portal_phase}"
printf 'TrainingPortal: %s\n' "${portal_runtime}"
printf 'Portal hosts:\n%s\n' "${portal_hosts}"

routing_namespaces=$("${kubectl[@]}" get package -A \
  -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\n"}{end}' \
  | awk '$2 == "educates-session" { print $1 }' || true)

if [[ -z ${routing_namespaces} ]]; then
  printf 'No session routing namespaces exist yet. Allocate a portal session, then rerun this check.\n'
  exit 0
fi

while IFS= read -r namespace; do
  [[ $("${kubectl[@]}" get package educates-session -n "${namespace}" \
    -o jsonpath='{.status.phase}') == Ready ]]
  "${kubectl[@]}" get virtualservice -n "${namespace}"
done <<<"${routing_namespaces}"
