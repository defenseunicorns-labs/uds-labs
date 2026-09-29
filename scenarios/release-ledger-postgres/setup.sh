#!/bin/bash
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
set -euo pipefail
export HOME=/root
mkdir -p /var/log/lab-setup /root/.kube
log() { echo "[$(date '+%H:%M:%S')] $*" | tee -a /var/log/lab-setup/uds-setup.log; }
systemctl start docker
sleep 3
k3d cluster start uds
k3d kubeconfig get uds > /root/.kube/config
chmod 600 /root/.kube/config
export KUBECONFIG=/root/.kube/config
mkdir -p /root/release-ledger-postgres
cp -a /opt/scenario/app/. /root/release-ledger-postgres/
{
  log "Waiting for UDS Core pods..."
  uds zarf tools kubectl wait --for=condition=Available deployment --all --all-namespaces --timeout=300s >> /var/log/lab-setup/uds-setup.log 2>&1 || true
  touch /var/log/lab-setup/pods-ready
  log "Building release-ledger:dev Docker image..."
  docker build -t release-ledger:dev /root/release-ledger-postgres --network host >> /var/log/lab-setup/uds-setup.log 2>&1
  touch /var/log/lab-setup/image-ready /var/log/lab-setup/ready
  log "Release Ledger PostgreSQL lab ready."
} &
