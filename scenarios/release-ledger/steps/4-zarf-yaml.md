# Step 4 – Write zarf.yaml

`zarf.yaml` defines the application package: the Helm chart and the container image. Cluster infrastructure and backing services belong in the bundle, not here.

```bash
cat > /root/release-ledger/zarf.yaml << 'EOF'
kind: ZarfPackageConfig
metadata:
  name: release-ledger
  description: "Stateless Release Ledger packaged for UDS"
  version: dev

components:
  - name: release-ledger
    required: true
    charts:
      - name: release-ledger
        localPath: chart/
        namespace: release-ledger
        version: 0.1.0
    images:
      - release-ledger:dev
EOF
```

Zarf bundles the image layers and Helm chart into a self-contained archive. The image does not need to be pulled from a registry during deployment.

The application package owns Release Ledger and its UDS integration. It does not own k3d, UDS Core, a database, or object storage.

## Verify

```bash
grep -q "ZarfPackageConfig" zarf.yaml && \
grep -q "release-ledger:dev" zarf.yaml && \
echo "zarf.yaml looks good"
```
