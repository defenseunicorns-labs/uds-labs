# Local development

This guide covers supported local development workflows and every task defined
by this repository. Run commands from the repository root unless stated
otherwise.

> [!CAUTION]
> `uds run dev` and `uds run cluster-up` default to `WIPE_CLUSTER=1`. They
> uninstall an existing k3s installation before rebuilding the development
> cluster. Use `--with WIPE_CLUSTER=0` to preserve the current cluster.

## Prerequisites

The full end-to-end environment requires:

- Bare-metal Linux with `/dev/kvm` and at least 80 GB of free disk
- Internet access for the first build
- `uds`, `zarf`, `kubectl`, `docker` with Compose, `curl`, `jq`, `yq`, and `ip`
- Go 1.26 for Go development
- `helm` for package validation and `golangci-lint` for `uds run lint`
- `virtctl` for VM console and SSH access
- Packer 1.9 or newer, `qemu-img`, and `qemu-system-x86_64` when building VM
  images
- Published KubeVirt UDS package access to `ghcr.io/uds-packages/kubevirt`
- Placement-enabled CDI checkout: `~/src/github.com/uds-packages/containerized-data-importer`

Override the CDI artifact when it lives elsewhere:

```bash
uds run preflight \
  --with cdi_package=/path/to/zarf-package-cdi-amd64-dev-unicorn.tar.zst
```

The Unicorn CDI flavor also requires authentication to the Defense Unicorns
Chainguard registry. The `upstream` flavor does not use that image source.

## Discover available tasks

List the 33 tasks maintained in this repository:

```bash
uds run --list
```

List repository tasks plus tasks imported from `udm-common` and `uds-common`:

```bash
uds run --list-all
```

Validate task syntax without running task commands:

```bash
uds run --dry-run <task>
```

Pass task inputs with `--with`:

```bash
uds run dev --with WIPE_CLUSTER=0 \
  --with CDI_PACKAGE="$HOME/src/github.com/uds-packages/containerized-data-importer/zarf-package-cdi-amd64-dev-unicorn.tar.zst"
```

Input names are case-sensitive.

## Full local E2E

The committed [`bundle/local/uds-bundle.yaml`](../bundle/local/uds-bundle.yaml)
profile is the supported bare-k3s profile. It uses empty Kubernetes placement,
`local-path` storage, `uds.dev`, capacity one, and no environment-specific
Keycloak group restriction.

Refresh sudo credentials immediately before a full run:

```bash
sudo -v
uds run local-e2e \
  --with CDI_PACKAGE="$HOME/src/github.com/uds-packages/containerized-data-importer/zarf-package-cdi-amd64-dev-upstream.tar.zst"
```

`uds run dev` is an alias for `local-e2e`. The task performs these steps in
order:

1. Preflight checks for KVM, tools, and external package artifacts.
2. Recreate bare k3s, install MetalLB, and initialize Zarf.
3. Deploy the pinned standalone UDS Core 1.10.0 `core-base` and
   `core-identity-authorization` Zarf packages directly. No k3d bundle or k3d
   package is used; the identity package installs both Keycloak and authservice.
4. Pull KubeVirt, stage CDI, and build the VM-image and UDS Labs packages.
5. Build and deploy the local bundle profile.
6. Patch cluster DNS, start the local SNI proxy, and create the test user.
7. Verify golden DataVolumes, the UDS Labs server, and authservice.

When complete:

- UI: `https://labs.uds.dev`
- Keycloak admin: `https://keycloak.admin.uds.dev`
- Test user: `doug@uds.dev`
- Test password: `unicorn123!@#UN`

Open the UI, authenticate, start a Lab, and verify terminal and browser access.
For an operator-only lifecycle check:

```bash
./scripts/create-test-session.sh
kubectl get labsession -A -w
kubectl get vmi -n uds-labs-vms -w
```

### VM-image inputs

By default, the task reuses a matching ignored package under
`packages/vm-images/`. If one is not present, it pulls the matching package
version from GHCR.

Build both qcow2 images and package them locally:

```bash
uds run local-e2e --with BUILD_IMAGES=1 --with LOCAL_VM_IMAGES=1
```

Reuse existing qcow2 outputs while rebuilding the local package:

```bash
uds run local-e2e --with BUILD_IMAGES=0 --with LOCAL_VM_IMAGES=1
```

Before reusing qcow2 files, verify them:

```bash
test -s packer/output/base/lab-base.qcow2
test -s packer/output/uds-core/lab-playground-uds-core.qcow2
docker info >/dev/null
```

### Preserve the cluster

Rebuild packages and redeploy while preserving k3s:

```bash
uds run dev --with WIPE_CLUSTER=0
```

For the fastest server or operator iteration, rebuild only the application
image and package:

```bash
uds run redeploy
uds run smoke-test
```

Watch the operator after redeployment:

```bash
kubectl logs -n uds-labs -l app=lab-operator -f
```

### Use locally built VM images

Build both qcow2 images, wrap them in local image-server containers, and embed
them in the package:

```bash
uds run dev --with BUILD_IMAGES=1 --with LOCAL_VM_IMAGES=1
```

Reuse existing qcow2 files while rebuilding the package:

```bash
uds run dev \
  --with WIPE_CLUSTER=0 \
  --with BUILD_IMAGES=0 \
  --with LOCAL_VM_IMAGES=1
```

Skip a tier only when its existing qcow2 output is present:

```bash
uds run build-images --with skip_base=1
uds run build-images --with skip_uds_core=1
```

## Validate changes

### Go code

```bash
go test ./...
go vet ./...
uds run lint
```

Run one package or test while iterating:

```bash
go test ./internal/controller
go test ./cmd/labserver -run TestName
```

### Helm and Zarf packaging

Run the repository's no-cluster validation:

```bash
uds run dry-run
```

This runs packaging contract tests, Helm lint and render, then Zarf lint and
manifest render.

Validate only the VM image-server and CDI import configuration:

```bash
uds run validate-package-config
```

### Deployed environment

```bash
uds run verify-cluster
uds run smoke-test
```

Apply the sample session and watch its resources:

```bash
./scripts/create-test-session.sh
kubectl get labsession -A -w
kubectl get vmi -n uds-labs-vms -w
```

## Run the server from source

The server needs a reachable Kubernetes API and the `LabSession` CRD. It is not
a standalone mock mode. After bringing up the cluster, run it against the
current kubeconfig:

```bash
DEV_MODE=true \
SCENARIOS_DIR=./scenarios \
STATIC_DIR=./web/static \
go run ./cmd/labserver
```

`DEV_MODE=true` grants admin access to authenticated users and must never be
used in production. The server listens on `:8080` by default.

Useful server environment variables:

| Variable | Default | Purpose |
|---|---:|---|
| `PORT` | `8080` | HTTP listen port |
| `VM_NAMESPACE` | `uds-labs-vms` | Namespace containing lab VM resources |
| `SERVER_NAMESPACE` | value of `VM_NAMESPACE` | Namespace containing server-owned resources |
| `SESSION_TTL_MINUTES` | `60` | Session lifetime |
| `SCENARIOS_DIR` | embedded scenarios | Read scenario files from disk |
| `STATIC_DIR` | embedded web assets | Read static assets from disk |
| `DEV_MODE` | `false` | Grant authenticated users development admin access |
| `DEMO_TOKEN_HMAC_KEY` | unset | Enable demo-token routes when at least 32 bytes |

## Develop the browser IDE

The IDE source has its own Vite project:

```bash
cd web/ide-src
npm ci
npm run dev
```

Build assets for the embedded application:

```bash
cd web/ide-src
npm run build
```

Commit generated assets under `web/static/ide-assets/` when the source change
requires them.

## Repository task reference

### Validation and cleanup

| Task | What it does | Important inputs or notes |
|---|---|---|
| `clean-artifacts` | Removes built Zarf packages, bundles, image tarballs, and the Zarf temp cache | `clean=1`, `skip=0`; destructive |
| `lint` | Runs `golangci-lint run ./...` | Requires `golangci-lint` |
| `preflight` | Checks required tools, KVM, CDI package artifact, and package configuration | `cdi_package` |
| `validate-package-config` | Checks VM image-server and CDI import contracts | No cluster required |
| `dry-run` | Runs tests and renders Helm and Zarf packages | No cluster required |

### VM and application builds

| Task | What it does | Important inputs or notes |
|---|---|---|
| `build-images` | Builds base then UDS Core qcow2 images with Packer | `build=1`, `skip_base=0`, `skip_uds_core=0`; up to two hours |
| `build-vm-images` | Builds local OCI image servers from qcow2 outputs | `build=1`, `base_qcow2`, `uds_core_qcow2`, `tag` |
| `push-vm-images` | Pushes base and UDS Core image-server images to GHCR | `tag`; requires registry write access |
| `build-image` | Builds the versioned uds-labs container image | Version comes from `zarf.yaml` |
| `push-image` | Pushes the uds-labs image to GHCR | Requires registry write access |
| `stage-cdi-package` | Stages a prebuilt external CDI package for bundle creation | `package` |
| `build-lab-package` | Builds the UDS Labs server/operator Zarf package | none |
| `build-vm-images-package` | Reuses/pulls a versioned VM-image package or builds one locally | `local`, `repository` |
| `build-bundle` | Creates the selected UDS bundle | `profile=default\|local`; requires dependency packages |

### Cluster infrastructure

| Task | What it does | Important inputs or notes |
|---|---|---|
| `wipe-k3s` | Uninstalls k3s when present | `run=1`, `skip=0`; destructive |
| `install-k3s` | Installs pinned k3s without Traefik or ServiceLB | `run=1`, `skip=0`; requires sudo |
| `install-metallb` | Installs MetalLB and configures a local address pool | `version=v0.14.9`, `range` auto-detected |
| `zarf-init` | Initializes Zarf, retrying the known injector cleanup race | `run=1`, `skip=0` |
| `deploy-local-uds-core` | Deploys standalone UDS Core base and identity packages on bare k3s | `run`, pinned `base_package`, `identity_package` |
| `wait-kubevirt` | Waits for KubeVirt to report Available | Existing cluster required |
| `populate-golden-pvcs` | Imports local qcow2 files through temporary HTTP servers | `base_qcow2`, `uds_core_qcow2`, `namespace`, `skip=0` |
| `cluster-up` | Lower-level k3s plus selected bundle deployment compatibility flow | Does not install UDS Core or configure browser access |
| `verify-cluster` | Prints node, KubeVirt, and workload status | Read-only |

### Deploy and operate

| Task | What it does | Important inputs or notes |
|---|---|---|
| `deploy-bundle` | Deploys the newest selected-profile bundle | `profile=default\|local`; run `build-bundle` first |
| `patch-demo-routes` | Restores unauthenticated demo-route exemptions | Called by `deploy-stack`; rerun after direct Package CR changes |
| `patch-coredns` | Maps UDS hostnames to MetalLB gateways and restarts authservice | Existing deployed cluster required |
| `create-test-user` | Creates the documented Keycloak test user | Calls shared `uds-setup:keycloak-user` |
| `start-proxy` | Starts the local nginx SNI proxy on ports 80 and 443 | Requires gateway LoadBalancer IPs |
| `stop-proxy` | Stops the local nginx SNI proxy | Uses Docker Compose |
| `redeploy` | Rebuilds and redeploys without rebuilding infrastructure packages | `BUNDLE_PROFILE=local`, `vm_image_tag` |
| `smoke-test` | Checks golden PVCs, app pod, authservice, and demo-route policy exemptions | Returns nonzero on failure |
| `local-e2e` | Runs the supported bare-k3s end-to-end workflow | Defaults to `WIPE_CLUSTER=1`; see inputs below |
| `dev` | Backward-compatible alias for `local-e2e` | Same inputs as `local-e2e` |

### Composite task inputs

`local-e2e` and `dev` accept:

| Input | Default | Effect |
|---|---:|---|
| `WIPE_CLUSTER` | `1` | Recreate k3s and UDS Core; use `0` to preserve them |
| `CDI_PACKAGE` | external upstream artifact | Select the prebuilt CDI package |
| `UDS_CORE_BASE_PACKAGE` | UDS Core `core-base:1.10.0-upstream` | Standalone base package for bare k3s |
| `UDS_CORE_IDENTITY_PACKAGE` | UDS Core `core-identity-authorization:1.10.0-upstream` | Standalone Keycloak and authservice package for bare k3s |
| `BUILD_IMAGES` | `0` | Build local qcow2 images |
| `LOCAL_VM_IMAGES` | `0` | Build the VM-image package locally instead of reusing/pulling it |
| `SKIP_BASE` | `0` | Skip rebuilding the base qcow2 when image building is enabled |
| `SKIP_UDS_CORE` | `0` | Skip rebuilding the UDS Core qcow2 when image building is enabled |
| `VM_IMAGE_TAG` | package version | Select the VM image-server tag |

`cluster-up` remains a lower-level compatibility task. It does not install UDS
Core or configure local browser access; prefer `local-e2e` for full testing.

## Shared included tasks

`tasks.yaml` imports pinned tasks from `udm-common` and `uds-common`. These are
mostly release, security, tool-install, and generic UDS Core helpers. Inspect
their current names and descriptions with:

```bash
uds run --list-all
```

Current namespaces:

- `attest:*` for attested linting
- `build:*` for attested Zarf package builds
- `olm:*` for OLM authentication and Fulcio tokens
- `publish:*` for OCI package publishing
- `scan:*` for Gitleaks and OpenGrep installation and scans
- `setup:*` for UDS CLI, Cosign, and Witness installation
- `vouch:*` for package attestations
- `uds-setup:*` for generic UDS Core test clusters and Keycloak users

These shared tasks can change when their pinned include versions change.
`uds run --list-all` is the authoritative runtime list.

## Troubleshooting

### The existing cluster disappeared

`dev` and `cluster-up` use `WIPE_CLUSTER=1` by default. Preserve an existing
cluster with:

```bash
uds run dev --with WIPE_CLUSTER=0
```

### Sudo expires during cluster setup

Refresh credentials immediately before the task:

```bash
sudo -v
uds run cluster-up
```

### UDS hostnames do not resolve

Redeployments can replace CoreDNS settings. Reapply them:

```bash
uds run patch-coredns
uds run start-proxy
```

### Demo routes redirect to SSO

`deploy-stack` restores route exemptions automatically after bundle deployment.
Pepr can still overwrite them after a direct Package CR change; restore and
verify them with:

```bash
uds run patch-demo-routes
uds run smoke-test
```

### Golden PVC imports fail

Inspect CDI resources and events:

```bash
kubectl get datavolumes -n uds-labs-vms
kubectl get pods -n cdi
kubectl get events -n uds-labs-vms --sort-by=.lastTimestamp
```

Use the local qcow2 fallback only when packaged image servers are unavailable:

```bash
uds run populate-golden-pvcs --with skip=0
```

### Proxy ports are already in use

Stop the repository proxy, then inspect port owners:

```bash
uds run stop-proxy
sudo ss -ltnp '( sport = :80 or sport = :443 )'
```

### Inspect a running VM

```bash
kubectl get vmi -n uds-labs-vms
virtctl console <vmi-name> -n uds-labs-vms
virtctl ssh --local-ssh-opts="-i $(pwd)/packer/packer-key" \
  lab@vmi/<vmi-name> -n uds-labs-vms
```
