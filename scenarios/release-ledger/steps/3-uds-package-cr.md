# Step 3 – The UDS Package CR: expose the app

The `Package` CR tells UDS Core how to integrate the application with the mesh and ingress. This baseline has no database, object store, or SSO, so the Package CR only needs application networking.

## Write the CR

```bash
cat > chart/templates/uds-package.yaml << 'EOF'
apiVersion: uds.dev/v1alpha1
kind: Package
metadata:
  name: release-ledger
  namespace: {{ .Release.Namespace }}
spec:
  network:
    serviceMesh:
      mode: ambient
    expose:
      - service: release-ledger
        selector:
          app: release-ledger
        gateway: tenant
        host: release-ledger
        port: 8080
        uptime:
          checks:
            - path: /health
              port: 8080
    allow:
      - direction: Ingress
        selector:
          app: release-ledger
        remoteGenerated: IntraNamespace
        description: Allow namespace-local ingress
      - direction: Egress
        selector:
          app: release-ledger
        remoteGenerated: IntraNamespace
        description: Allow namespace-local egress
EOF
```

## Understand the configuration

- `serviceMesh.mode: ambient` uses UDS Core's node-level mesh.
- `expose` creates the route from `release-ledger.uds.dev` to the Service.
- The selector must match both the Deployment labels and the Service selector.
- The uptime check uses the application's `/health` endpoint.
- The namespace-local rules allow the application to participate in the local mesh while retaining UDS Core's default-deny posture.

There is intentionally no SSO block in this baseline. Authentication is taught separately.

## Verify

```bash
grep -q "kind: Package" chart/templates/uds-package.yaml && \
grep -q "network:" chart/templates/uds-package.yaml && \
grep -q "expose:" chart/templates/uds-package.yaml && \
echo "Package CR looks good"
```
