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
- Local KubeVirt package checkout: `~/src/github.com/uds-packages/kubevirt`
- Prebuilt CDI package artifact from the separate CDI repository

Override either checkout when it lives elsewhere:

```bash
uds run preflight \
  --with kubevirt_pkg_dir=/path/to/kubevirt \
  --with cdi_package=/path/to/zarf-package-cdi-amd64-dev-unicorn.tar.zst
```

The Unicorn CDI flavor also requires authentication to the Defense Unicorns
Chainguard registry. The `upstream` flavor does not use that image source.

## Discover available tasks

List the 31 tasks maintained in this repository:

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

## Start a development environment

Refresh sudo credentials before a full cluster build. The k3s steps require
sudo and may run long enough for an existing sudo session to expire.

```bash
sudo -v
uds run dev --with CDI_PACKAGE="$HOME/src/github.com/uds-packages/containerized-data-importer/zarf-package-cdi-amd64-dev-unicorn.tar.zst"
```

The workflow builds dependencies, creates or replaces the k3s cluster, deploys
the platform, patches local DNS and auth routes, starts the local SNI proxy,
creates the test user, and runs smoke tests.

When complete:

- UI: `https://lab.uds.dev`
- Keycloak admin: `https://keycloak.admin.uds.dev`
- Test user: `doug@uds.dev`
- Test password: `unicorn123!@#UN`

### Nuclear reset with existing local VM images

Use this flow to destroy and recreate the local k3s cluster, rebuild the
Unicorn CDI package, and deploy image-server containers from existing local
qcow2 files:

```bash
uds run dev \
  --with CDI_PACKAGE="$HOME/src/github.com/uds-packages/containerized-data-importer/zarf-package-cdi-amd64-dev-unicorn.tar.zst" \
  --with WIPE_CLUSTER=1 \
  --with BUILD_IMAGES=0 \
  --with LOCAL_VM_IMAGES=1
```

This is the nuclear option:

- `WIPE_CLUSTER=1` removes the existing k3s cluster and cleans generated
  packages, bundles, image tarballs, and the Zarf temporary cache.
- `CDI_PACKAGE` selects a prebuilt CDI artifact. Build the Unicorn flavor in
  the separate CDI repository first.
- `BUILD_IMAGES=0` skips the hour-long Packer build. It does not create qcow2
  files.
- `LOCAL_VM_IMAGES=1` wraps existing qcow2 files in local image-server
  containers and includes them in the deployment.
- `GITHUB_TOKEN="$(gh auth token)"` supplies the current GitHub CLI token only
  to this command and its child processes. Do not run it with shell tracing
  enabled.

Before running it, verify authentication, local images, Docker, and sudo:

```bash
gh auth status
test -s packer/output/base/lab-base.qcow2
test -s packer/output/uds-core/lab-playground-uds-core.qcow2
docker info >/dev/null
sudo -v
```

If either qcow2 check fails, build the missing images first or change
`BUILD_IMAGES` to `1`. The `build-vm-images` task skips a tier whose qcow2
directory is empty, so these checks prevent a late provisioning failure.

The `dev` task ends by running `smoke-test`. A successful run reports the UI
and Keycloak URLs. Recheck the deployed state at any time with:

```bash
uds run smoke-test
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
kubectl logs -n lab-platform -l app=lab-operator -f
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
kubectl apply -f test-session.yaml
kubectl get labsession -A -w
kubectl get vmi -n uds-lab-vms -w
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
| `VM_NAMESPACE` | `uds-lab-vms` | Namespace containing lab VM resources |
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
| `preflight` | Checks required tools, KVM, package artifacts, and package configuration | `kubevirt_pkg_dir`, `cdi_package` |
| `validate-package-config` | Checks VM image-server and CDI import contracts | No cluster required |
| `dry-run` | Runs tests and renders Helm and Zarf packages | No cluster required |

### VM and application builds

| Task | What it does | Important inputs or notes |
|---|---|---|
| `build-images` | Builds base then UDS Core qcow2 images with Packer | `build=1`, `skip_base=0`, `skip_uds_core=0`; up to two hours |
| `build-vm-images` | Builds local OCI image servers from qcow2 outputs | `build=1`, `base_qcow2`, `uds_core_qcow2`, `tag` |
| `push-vm-images` | Pushes base and UDS Core image-server images to GHCR | `tag`; requires registry write access |
| `build-image` | Builds the versioned lab-platform container image | Version comes from `zarf.yaml` |
| `push-image` | Pushes the lab-platform image to GHCR | Requires registry write access |
| `build-kubevirt` | Builds the external KubeVirt Zarf package | `dir`, `rebuild=1`, `skip_if_exists=0` |
| `stage-cdi-package` | Stages a prebuilt external CDI package for bundle creation | `package` |
| `build-lab-package` | Builds the lab-platform Zarf package | `vm_image_tag`; defaults to package version |
| `build-bundle` | Creates the UDS bundle under `bundle/` | Requires access to internal package dependencies |

### Cluster infrastructure

| Task | What it does | Important inputs or notes |
|---|---|---|
| `wipe-k3s` | Uninstalls k3s when present | `run=1`, `skip=0`; destructive |
| `install-k3s` | Installs pinned k3s without Traefik or ServiceLB | `run=1`, `skip=0`; requires sudo |
| `install-metallb` | Installs MetalLB and configures a local address pool | `version=v0.14.9`, `range` auto-detected |
| `zarf-init` | Initializes Zarf, retrying the known injector cleanup race | `run=1`, `skip=0` |
| `wait-kubevirt` | Waits for KubeVirt to report Available | Existing cluster required |
| `populate-golden-pvcs` | Imports local qcow2 files through temporary HTTP servers | `base_qcow2`, `uds_core_qcow2`, `namespace`, `skip=0` |
| `cluster-up` | Runs the complete infrastructure and package deployment sequence | Defaults to `WIPE_CLUSTER=1`; see inputs below |
| `verify-cluster` | Prints node, KubeVirt, and workload status | Read-only |

### Deploy and operate

| Task | What it does | Important inputs or notes |
|---|---|---|
| `deploy-bundle` | Deploys the newest bundle tarball from `bundle/` | Run `build-bundle` first |
| `patch-demo-routes` | Restores unauthenticated demo-route exemptions | `namespace=lab-platform`; rerun after Helm reconciliation |
| `patch-coredns` | Maps UDS hostnames to MetalLB gateways and restarts authservice | Existing deployed cluster required |
| `create-test-user` | Creates the documented Keycloak test user | Calls shared `uds-setup:keycloak-user` |
| `start-proxy` | Starts the local nginx SNI proxy on ports 80 and 443 | Requires gateway LoadBalancer IPs |
| `stop-proxy` | Stops the local nginx SNI proxy | Uses Docker Compose |
| `redeploy` | Rebuilds and redeploys only the application package | `vm_image_tag` |
| `smoke-test` | Checks golden PVCs, demo auth policies, app pod, and authservice | Returns nonzero on failure |
| `dev` | Runs the full local end-to-end workflow | Defaults to `WIPE_CLUSTER=1`; see inputs below |

### Composite task inputs

`dev` accepts:

| Input | Default | Effect |
|---|---:|---|
| `WIPE_CLUSTER` | `1` | Uninstall and reinstall k3s |
| `BUILD_IMAGES` | `0` | Build local qcow2 images |
| `LOCAL_VM_IMAGES` | `0` | Build and embed local VM image-server containers |
| `SKIP_BASE` | `0` | Reuse the existing base qcow2 |
| `SKIP_UDS_CORE` | `0` | Reuse the existing UDS Core qcow2 |
| `SKIP_GOLDEN_PVC` | `1` | Skip the host-served qcow2 fallback |
| `VM_IMAGE_TAG` | package version | Select the VM image-server tag |
| `CDI_PACKAGE` | CDI package artifact under the external checkout | Select the prebuilt CDI package |

`cluster-up` accepts the same inputs plus:

| Input | Default | Effect |
|---|---:|---|
| `KUBEVIRT_PKG_DIR` | `~/src/github.com/uds-packages/kubevirt` | Select the KubeVirt checkout |

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

Pepr can overwrite route exemptions during Package CR reconciliation:

```bash
uds run patch-demo-routes
uds run smoke-test
```

### Golden PVC imports fail

Inspect CDI resources and events:

```bash
kubectl get datavolumes -n uds-lab-vms
kubectl get pods -n cdi
kubectl get events -n uds-lab-vms --sort-by=.lastTimestamp
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
kubectl get vmi -n uds-lab-vms
virtctl console <vmi-name> -n uds-lab-vms
virtctl ssh --local-ssh-opts="-i $(pwd)/packer/packer-key" \
  lab@vmi/<vmi-name> -n uds-lab-vms
```
