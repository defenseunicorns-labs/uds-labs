# Step 1 – Choose the auth boundary

```bash
cd /root/secure-release-ledger
sed -n '1,220p' app.py
```

Release Ledger has no login handler, session middleware, or OIDC token validation. This scenario therefore uses UDS Authservice to protect the workload at the platform boundary. Do not add native OIDC configuration to the image.

Inspect the Deployment and Service labels before editing the Package CR:

```bash
grep -n -A12 -B4 'labels:\|selector:' chart/templates/deployment.yaml chart/templates/service.yaml
```

## Verify

```bash
grep 'def create_app' app.py
grep 'kind: Service' chart/templates/service.yaml
grep 'app: secure-release-ledger' chart/templates/deployment.yaml
echo 'Auth boundary identified'
```
