# Secure an Existing App with UDS SSO

Release Ledger has no native login flow. Protect it with UDS Authservice instead of adding application OIDC code.

The UDS Package CR registers a Keycloak client and tells UDS Core which pods to protect. The selector must match both the Deployment pod labels and the Service selector. Authservice redirect URIs must use a specific path such as `/login`, not the host root.

All work lives in `/root/secure-release-ledger`. Do not edit `app/app.py` or patch live resources.
