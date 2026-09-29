# Step 3 – Trace selectors and routing

Authservice must select the same pods reached by the exposed Service. Compare all three declarations:

```bash
cd /root/secure-release-ledger
grep -n -A8 -B3 'app: secure-release-ledger' chart/templates/deployment.yaml chart/templates/service.yaml chart/templates/uds-package.yaml
```

The selector is `app: secure-release-ledger` in pod labels, Service `spec.selector`, and `enableAuthserviceSelector`. A mismatch can produce a Package that reconciles but a route that returns 503.

The exposure owns the stable host prefix `secure-release-ledger`; the deployment domain supplies `.uds.dev`. Keep the Service port and exposure port consistent.

## Verify

```bash
awk '/enableAuthserviceSelector:/{found=1} found{print} /network:/{exit}' chart/templates/uds-package.yaml | grep 'app: secure-release-ledger'
grep -A4 'selector:' chart/templates/service.yaml | grep 'app: secure-release-ledger'
grep 'host: secure-release-ledger' chart/templates/uds-package.yaml
grep 'port: 8080' chart/templates/uds-package.yaml
echo 'Selector and route contract aligned'
```
