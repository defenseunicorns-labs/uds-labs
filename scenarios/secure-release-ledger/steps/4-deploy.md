# Step 4 – Deploy and troubleshoot

Build and deploy the package from source:

```bash
cd /root/secure-release-ledger
uds run dev
```

Do not treat the command exit status as the health signal. Inspect the Package status and generated resources:

```bash
uds zarf tools kubectl get package secure-release-ledger -n secure-release-ledger -o yaml
uds zarf tools kubectl get pods -n secure-release-ledger
uds zarf tools kubectl get requestauthentication,authorizationpolicy -n secure-release-ledger
uds zarf tools kubectl get virtualservice -A | grep secure-release-ledger
```

If the Package is not Ready, inspect events and the UDS Operator logs. If the route returns 503, re-check the selector against both the Deployment and Service. If the redirect is rejected, re-check that it is `/login` rather than a root path.

## Verify

```bash
export KUBECONFIG=/root/.kube/config
uds zarf tools kubectl get package secure-release-ledger -n secure-release-ledger >/dev/null
uds zarf tools kubectl get deployment secure-release-ledger -n secure-release-ledger >/dev/null
echo 'Protected package deployed'
```
