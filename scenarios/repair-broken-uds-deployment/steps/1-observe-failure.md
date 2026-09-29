# Step 1 – Observe the failure

You inherited a package that was reported as “working.” Start by reproducing the deployment and
collecting evidence before editing anything.

```bash
cd /root/release-ledger
timeout --foreground 90s uds run dev
```

The deployment is expected to fail or time out. That is intentional; continue to the inspection
commands rather than retrying the deployment.

The first deployment may fail or appear successful while the application remains unhealthy. The
short timeout is intentional: it returns control for diagnosis while leaving the deployed resources
available for inspection. That
is useful evidence, not a reason to retry the same command.

Inspect the workload:

```bash
uds zarf tools kubectl get pods -n release-ledger
```

Inspect pod events and scheduling:

```bash
uds zarf tools kubectl describe pod -n release-ledger -l app=release-ledger
```

Inspect application logs:

```bash
uds zarf tools kubectl logs -n release-ledger -l app=release-ledger --all-containers
```

Render the package:

```bash
uds zarf tools helm template release-ledger chart/
```

Look for the first causal failure. Check whether the pod can start, whether it can be scheduled, and
whether the rendered Service and Package CR point at the same workload. There are several independent
configuration defects in this handoff; fixing the first visible symptom is not the end of the review.

Use these boundaries to organize the evidence:

| Evidence | Inspect |
| --- | --- |
| Pod is `Pending` | Deployment requests and available lab capacity. |
| Pod is restarting or cannot start | Image user, writable paths, and mounted volumes. |
| Service has no useful endpoint | Service selector, port, and target port. |
| Package is Ready but the route fails | Package exposure service, host, port, and generated resources. |
| Policy is present but does not cover the workload | Package CR selectors, direction, remote target, and port. |

If an older package produces `expected map got &{[map` for `spec.network.expose[].uptime`, use the
current UDS Core shape: `uptime.checks.paths` is a list of paths. The uptime configuration does not
use a list of `{path, port}` maps.

Do not edit `app/app.py`. The package must adapt to the application contract without changing the
application source.

## Verify

The initial fixture contains the submitted defects. Confirm that you inspected the files before
making changes:

```bash
test -f chart/templates/deployment.yaml
```

```bash
test -f chart/templates/service.yaml
```

```bash
test -f chart/templates/uds-package.yaml
```

```bash
grep 'cpu: "4"' chart/templates/deployment.yaml
```

```bash
grep 'targetPort: 8081' chart/templates/service.yaml
```

```bash
grep 'release-ledger-api' chart/templates/uds-package.yaml
```

```bash
grep 'customer.example.com' chart/templates/uds-package.yaml
```

The initial package inspection is complete when each command succeeds.
