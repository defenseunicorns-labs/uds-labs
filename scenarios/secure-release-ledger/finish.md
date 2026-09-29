# Lab Complete

Release Ledger is protected by UDS Authservice.

You verified the two-sided platform contract:

- The Package CR registers the client and Authservice selector.
- The selector matches both pods and Service.
- The redirect URI uses a valid non-root path.
- UDS Core creates the Keycloak/Authservice/Istio resources.
- Unauthenticated traffic redirects and authenticated traffic reaches the app.

The application remained unchanged; authentication is a platform integration.
