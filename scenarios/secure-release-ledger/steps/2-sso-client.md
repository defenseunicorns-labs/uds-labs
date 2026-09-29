# Step 2 – Register the Authservice client

Inspect the UDS Package CR:

```bash
cd /root/secure-release-ledger
sed -n '1,100p' chart/templates/uds-package.yaml
```

The SSO entry must include:

- a stable `clientId`;
- a redirect URI with a specific path (`/login`, not `/`);
- `enableAuthserviceSelector` matching the workload.

UDS Core creates the Keycloak client and Authservice resources from this declaration. Do not create a Red Hat Keycloak CRD or apply a manual client.

Render the chart and inspect the result:

```bash
uds zarf tools helm template secure-release-ledger chart/ | sed -n '/kind: Package/,$p'
```

## Verify

```bash
grep 'sso:' chart/templates/uds-package.yaml
grep 'clientId: uds-secure-release-ledger' chart/templates/uds-package.yaml
grep 'redirectUris:' chart/templates/uds-package.yaml
grep '/login' chart/templates/uds-package.yaml
grep 'enableAuthserviceSelector:' chart/templates/uds-package.yaml
! grep 'https://secure-release-ledger\.uds\.dev$' chart/templates/uds-package.yaml
echo 'Authservice client configured'
```
