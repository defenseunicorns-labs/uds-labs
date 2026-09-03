#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
# Setup for the stateless Release Ledger scenario.
set -euo pipefail
export HOME=/root
mkdir -p /var/log/lab-setup /root/.kube

log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a /var/log/lab-setup/uds-setup.log; }

log "Starting Docker..."
systemctl start docker
sleep 3

log "Starting k3d cluster..."
k3d cluster start uds

log "Writing kubeconfig..."
k3d kubeconfig get uds > /root/.kube/config
chmod 600 /root/.kube/config
export KUBECONFIG=/root/.kube/config

log "Scaffolding /root/release-ledger..."
mkdir -p /root/release-ledger/chart/templates /root/release-ledger/bundle
cp -a /opt/scenario/app/. /root/release-ledger/

cat > /root/release-ledger/chart/values.yaml << 'EOF'
image: release-ledger:dev
domain: uds.dev
EOF

log "Scaffold complete."

{
  log "Waiting for UDS Core pods..."
  uds zarf tools kubectl wait --for=condition=Available deployment \
    --all --all-namespaces --timeout=300s >> /var/log/lab-setup/uds-setup.log 2>&1 || true
  touch /var/log/lab-setup/pods-ready
  log "UDS Core pods ready."

  log "Building release-ledger:dev Docker image..."
  docker build -t release-ledger:dev /root/release-ledger --network host >> /var/log/lab-setup/uds-setup.log 2>&1
  touch /var/log/lab-setup/image-ready
  log "release-ledger:dev built — lab fully ready."
  touch /var/log/lab-setup/ready
} &
