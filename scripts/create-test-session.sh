#!/usr/bin/env bash
set -euo pipefail

namespace="${NAMESPACE:-uds-labs-vms}"
scenario_id="${SCENARIO_ID:-playground-tools}"
client_id="${CLIENT_ID:-testuser}"
ttl_minutes="${TTL_MINUTES:-60}"
name="${NAME:-test-s$(date -u +%s)}"
expires_at="$(date -u -d "+${ttl_minutes} minutes" +%Y-%m-%dT%H:%M:%SZ)"

cat <<YAML | kubectl apply -f -
apiVersion: labs.uds.dev/v1alpha1
kind: LabSession
metadata:
  name: ${name}
  namespace: ${namespace}
spec:
  sessionID: ${name}
  scenarioID: ${scenario_id}
  clientID: ${client_id}
  expiresAt: "${expires_at}"
  size: small
YAML
