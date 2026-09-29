# Step 5 – Rebuild and verify

The source now describes a repeatable deployment. Move to the repaired package workspace:

```bash
cd /root/release-ledger
```

The first diagnostic deployment may still be registered with UDS Core. Remove that deployed
bundle before creating and deploying the repaired one:

```bash
uds remove bundle/uds-bundle-release-ledger-repair-amd64-dev.tar.zst --confirm --no-progress
```

Then build, bundle, and deploy the repaired source:

```bash
uds run dev
```

This creates two related artifacts:

- The **Zarf package** is the atomic deployable `.tar.zst` artifact for Release Ledger. It contains
  the Helm chart and embedded application image.
- The **UDS bundle** is the deployable composition that contains the application package and its
  deployment configuration.

During deployment, Zarf loads the embedded image into the cluster's internal registry. The workload
pulls the image from that internal registry rather than from the public internet.

Check the rollout:

```bash
uds zarf tools kubectl get pods -n release-ledger
```

Inspect the generated Package resource:

```bash
uds zarf tools kubectl get package release-ledger -n release-ledger -o yaml
```

Verify the health endpoint through the expected UDS hostname:

```bash
curl -skf https://release-ledger.uds.dev/health
```

Create a release through ingress:

```bash
curl -sk -X POST https://release-ledger.uds.dev/api/releases -H 'Content-Type: application/json' -d '{"application_name":"ledger-api","version":"1.0.0","environment":"staging"}'
```

Read releases through ingress:

```bash
curl -skf https://release-ledger.uds.dev/api/releases
```

## Verify

The scenario verifier checks that the package and bundle artifacts exist, the Deployment is
available without unexpected restarts, and the application responds through ingress.
