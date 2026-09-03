# Step 5 – Bundle and tasks

The UDS bundle is the deployment unit. It points to the application package and provides the place where future scenarios compose Postgres and MinIO dependencies.

## Bundle

```bash
cat > /root/release-ledger/bundle/uds-bundle.yaml << 'EOF'
kind: UDSBundle
metadata:
  name: release-ledger-bundle
  description: Bundle deploying the stateless Release Ledger application
  version: dev

packages:
  - name: release-ledger
    path: ../
    ref: dev
EOF
```

`path: ../` points to the parent directory, where `uds create` finds the package archive.

## Tasks

```bash
cat > /root/release-ledger/tasks.yaml << 'EOF'
# yaml-language-server: $schema=https://raw.githubusercontent.com/defenseunicorns/uds-cli/refs/heads/main/tasks.schema.json
includes:
  - setup: https://raw.githubusercontent.com/defenseunicorns/uds-common/v1.25.0/tasks/setup.yaml

tasks:
  - name: dev
    description: Build the package, create the bundle, and deploy it
    actions:
      - cmd: uds zarf package create . --confirm --skip-sbom --no-progress
      - cmd: uds create bundle/ --confirm --no-progress
      - cmd: uds deploy bundle/uds-bundle-release-ledger-bundle-amd64-dev.tar.zst --confirm --no-progress
EOF
```

The package owns the application. The bundle owns composition. The task gives the developer one repeatable build → bundle → deploy workflow.

## Verify

```bash
grep -q "UDSBundle" bundle/uds-bundle.yaml && \
grep -q "uds-common" tasks.yaml && \
grep -q "name: dev" tasks.yaml && \
echo "Bundle and tasks ready"
```
