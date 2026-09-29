#!/bin/bash
set -euo pipefail
cd "${SCENARIO_WORKSPACE:-/root/release-ledger-postgres}"
grep -q 'kind: ConfigMap' chart/templates/configmap.yaml
grep -q 'configMapKeyRef' chart/templates/deployment.yaml
grep -q 'secretKeyRef' chart/templates/deployment.yaml
# The chart keeps the Secret key names configurable in values.yaml; verify the
# references and their contract rather than requiring Helm's rendered literal.
grep -q 'key: {{ .Values.storage.usernameKey | quote }}' chart/templates/deployment.yaml
grep -q 'key: {{ .Values.storage.passwordKey | quote }}' chart/templates/deployment.yaml
grep -q '^  usernameKey: username$' chart/values.yaml
grep -q '^  passwordKey: password$' chart/values.yaml
! grep -R -Eq 'password:[[:space:]]*[^" ]' chart bundle
echo 'Workload credential wiring configured'
