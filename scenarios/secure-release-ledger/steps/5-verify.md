# Step 5 – Prove protected access

An unauthenticated request should redirect to the SSO issuer:

```bash
curl -skI https://secure-release-ledger.uds.dev/ | sed -n '1,12p'
```

Look for a 302/303 response and a `Location` containing the Keycloak SSO host. Do not use `curl -k https://.../health` as proof of access; that follows neither the browser session nor the authentication flow.

Before logging in, create the lab's Keycloak test user:

```bash
cd /root/secure-release-ledger && uds run setup:keycloak-user --with group="/UDS Core/Admin"
```

This creates the documented test identity `doug@uds.dev` with password
`unicorn123!@#UN`.

Open the application by clicking the **secure-release-ledger.uds.dev/** link in the service chip or service
list below. Complete the Keycloak login, then reload the application and confirm the Release Ledger
page is visible. Finally inspect the Package status and Authservice-related resources once more.

## Verify

```bash
curl -skI https://secure-release-ledger.uds.dev/ | grep -Eq 'HTTP/.*30[123]'
uds zarf tools kubectl get package secure-release-ledger -n secure-release-ledger -o jsonpath='{.status.phase}' | grep -q Ready
echo 'Authservice protection verified'
```
