# UDS Labs

Browser-based interactive lab environment for UDS and Zarf. Provisions ephemeral KubeVirt VMs on demand from golden PVC clones, serves browser terminals via ttyd, and requires no client installs.

## Architecture

```
Browser → Istio (TLS) → authservice (OIDC) → uds-labs server
                                                    │
                                          uds-labs operator
                                                    │
                                         ┌──────────┴──────────┐
                                         │     uds-labs-vms ns  │
                                    VMI (KubeVirt)              │
                                    DataVolume (CDI clone)      │
                                    Headless Service            │
                                    NetworkPolicy               │
                                         └─────────────────────┘

Lab VM (boots from golden PVC)
  ├── ttyd :7681   — tmux main session (setup-aware entry)
  ├── ttyd :7682   — direct bash shell
  ├── Python :7680 — lab-inject.py (cmd, verify, navigate, services)
  └── noVNC :6080  — Xvfb + x11vnc + websockify + Chromium (browser: true)
```

### Golden PVCs

VM images are built once with Packer (QEMU/KVM), wrapped in small Python HTTP
server images, and bundled by Zarf. Zarf rewrites the server Pods' normal image
references and supplies registry credentials at deploy time. CDI imports the
qcow2 files from stable cluster-local Services; no Zarf registry address appears
in a DataVolume. Each LabSession then clones the appropriate golden PVC, giving
every user an isolated copy of the full disk image. Pausing stops VM compute while
retaining that Session PVC; resume reattaches the same disk.

| Tier | Golden PVC | Contents |
|------|-----------|----------|
| `base` | `golden-base` | Ubuntu 24.04 + Docker, k3d, uds CLI, neovim, jq, yq, tmux, ttyd, noVNC, Chromium |
| `uds-core` | `golden-uds-core` | Base + k3d-core-slim-dev fully deployed |

## Prerequisites

**Host machine:**
- Bare-metal Linux with `/dev/kvm` (AMD-V or Intel VT-x enabled in BIOS)
- 80+ GB free disk for packer output
- `uds`, `zarf`, `kubectl`, `docker`, `jq`, `ip`, `curl`
- [virtctl](https://kubevirt.io/user-guide/user_workloads/virtctl_client_tool/) (for VM console/SSH access)
- Published KubeVirt UDS package access to `ghcr.io/uds-packages/kubevirt`
- Published CDI UDS package access to `~/src/github.com/uds-packages/containerized-data-importer`

**First-time only:**
- Internet access (pulls Ubuntu cloud image and standalone UDS Core packages)

## Releases

Releases follow the standard UDS Package Kit flow. The `upstream` release
coordinate is rendered into every repository-owned Labs artifact: the Zarf
package, `uds-labs` container image, Helm chart, and bundle. VM-image packages
retain an independent version because their Packer build and publication cadence differ.

1. Update the `upstream` version in [`releaser.yaml`](./releaser.yaml) through a pull request.
2. Run the [`release.yaml`](./.github/workflows/release.yaml) workflow manually from GitHub Actions.
3. Package Kit verifies and renders the release coordinate; `uds run release-tasks:prepare` then synchronizes the Labs package, image, chart, and bundle.
4. The workflow attests linting and security scans, builds the matching container image and one signed Zarf package, and validates that archive on k3d.
5. It publishes the validated archive to GHCR and creates the GitHub release/tag, vouches its attestations to CAT, then publishes the exact same archive to UDS Proving Ground.

The package is first published at `ghcr.io/defenseunicorns-labs/uds/uds-labs`
with an `-upstream` tag, then promoted to `registry.uds-mil.us/defenseunicorns`.
VM-image package publication remains a separate infrastructure artifact because
it requires the Packer/KVM build environment. It is published by the manual
[`Release VM Images`](./.github/workflows/release-vm-images.yaml) workflow, which
builds the Packer images, pushes the image-server containers, and publishes the
versioned Zarf package to `ghcr.io/defenseunicorns-labs/uds-labs-vm-images`.

The VM package uses the version in `packages/vm-images/zarf.yaml` as its release
coordinate; no second `releaser.yaml` is needed. `release-tasks:prepare` applies
only to the main UDS Labs package and does not update the VM package reference.
Bump the VM package version, the matching versions in its chart and bundle
reference, and (when image contents change) the `imageTag` and image archive
tags. The VM workflow verifies these references before building, then publishes
the package. Application releases continue to reuse the last published VM
package.

## Quick Start

### Full local E2E from scratch

The canonical bundle at [`bundle/uds-bundle.yaml`](bundle/uds-bundle.yaml)
uses published KubeVirt and CDI packages with portable placement defaults. The
ignored `bundle/uds-config.yaml` supplies environment-specific deployment values.

```bash
# sudo is needed for k3s uninstall/install
sudo -v
uds run local-e2e
```

`uds run local-e2e` performs the full KubeVirt flow:

1. Wipes and recreates bare k3s by default.
2. Installs MetalLB and initializes Zarf.
3. Deploys the standalone UDS Core 1.10.0 base and identity Zarf packages
   directly onto bare k3s.
4. Deploys the canonical bundle using published KubeVirt and CDI packages.
5. Reuses the staged VM-image package, or pulls the matching version from GHCR.
6. Patches CoreDNS, starts the local TLS proxy, and creates the test user.
7. Runs infrastructure and LabSession lifecycle smoke tests.

For the standard k3d package-development loop, use `uds run dev`. It validates
package installation and upgrades with the KubeVirt operator disabled because
k3d does not provide the KubeVirt/CDI APIs.

After completion:

- UI: `https://labs.uds.dev`
- Admin: `https://keycloak.admin.uds.dev`
- Test user: `doug@uds.dev / unicorn123!@#UN`

Open the UI, sign in, start a Lab, and verify the terminal/browser experience.
For an operator-only check, create a Session directly with
`uds run create-test-session`.

### VM-image choices

The default reuses a matching package already staged under `packages/vm-images/`.
If none exists, it attempts to pull the versioned package from
`ghcr.io/defenseunicorns-labs/uds-labs-vm-images`.

Build both qcow2 images and package them locally:

```bash
uds run local-e2e --with BUILD_IMAGES=1 --with LOCAL_VM_IMAGES=1
```

Reuse existing qcow2 outputs while rebuilding the local package:

```bash
uds run local-e2e --with BUILD_IMAGES=0 --with LOCAL_VM_IMAGES=1
```

### Preserve or redeploy the local cluster

```bash
uds run local-e2e --with WIPE_CLUSTER=0
uds run redeploy
```

With `WIPE_CLUSTER=0`, k3s and UDS Core are preserved. `redeploy` uses the
canonical bundle and rebuilds only the application/package stack.

## VM Images (Packer)

Images are built locally with QEMU/KVM and output as qcow2 files in `packer/output/`.

```bash
# Build both images
uds run build-images

# Skip specific tiers (reuse existing qcow2s)
uds run build-images --with skip_base=1
```

Build order: `lab-base` → `playground-uds-core`. The base image includes the
tools previously provided by a separate image. Each stage uses the previous
stage's qcow2 as its base disk. The UDS Core image takes ~45 min
(deploys a full k3d UDS Core cluster inside the VM before snapshotting).

### Import golden PVCs directly from qcow2 files (fallback)

The normal bundle deployment imports from the packaged image-server Services.
Use this host-served path only as a troubleshooting fallback:

```bash
uds run populate-golden-pvcs \
  --with base_qcow2=packer/output/base/lab-base.qcow2 \
  --with uds_core_qcow2=packer/output/uds-core/lab-playground-uds-core.qcow2
```

## Local Development

See [Local development](docs/local-development.md) for prerequisites, common
iteration loops, task inputs, troubleshooting, and the complete repository task
reference.

Discover tasks from the root entrypoint (implementations live under `tasks/`):

```bash
uds run --list       # repository tasks
uds run --list-all   # repository tasks plus imported shared tasks
```

Common workflows:

```bash
uds run dry-run                            # tests and package rendering; no cluster
uds run local-e2e                          # full bare-k3s browser E2E
uds run local-e2e --with WIPE_CLUSTER=0    # preserve k3s and UDS Core
uds run redeploy                           # fastest deployed-code iteration
uds run smoke-test
```

> **Warning:** `uds run local-e2e` and `uds run cluster-up` default to
> `WIPE_CLUSTER=1` and uninstall an existing k3s cluster.

For a clean cluster rebuild that reuses existing local qcow2 images, follow the
[nuclear reset workflow](docs/local-development.md#nuclear-reset-with-existing-local-vm-images).

## VM Access

```bash
# List running VMs
kubectl get vmi -n uds-labs-vms

# Serial console (shows cloud-init / user-data output)
virtctl console <vmi-name> -n uds-labs-vms   # exit: Ctrl+]

# SSH
virtctl ssh --local-ssh-opts="-i $(pwd)/packer/packer-key" \
  lab@vmi/<vmi-name> -n uds-labs-vms
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VM_NAMESPACE` | `uds-labs-vms` | Namespace for VMIs, DataVolumes, Services |
| `SESSION_TTL_MINUTES` | `60` | Lab session lifetime |
| `PORT` | `8080` | HTTP listen port |
| `SCENARIOS_DIR` | *(embedded)* | Override embedded scenarios with a local directory |
| `STATIC_DIR` | *(embedded)* | Override embedded static files |
| `MAX_ACTIVE_SESSIONS` | 1 | Number of lab sessions/vms that can be run at a time |

## Creating a Scenario

Scenarios live in `scenarios/<id>/`. Each directory needs:

```
scenarios/my-scenario/
├── scenario.yaml
├── setup.sh
├── steps/
│   ├── step1.md
│   └── step2.md
└── verify/           (optional)
    ├── step1.sh
    └── step2.sh
```

### scenario.yaml

```yaml
title: "My Scenario"
description: "What the learner will do."
outcome: "The capability the learner can demonstrate afterward."
prerequisites:
  - "Basic Kubernetes familiarity"
duration: 45
difficulty: beginner  # beginner | intermediate | advanced
browser: false        # true = provision Chromium + noVNC
tier: tools           # base | tools | uds-core — selects which golden PVC to clone
size: medium
orientation:
  mission: "The capability this lab helps the learner demonstrate."
  why: "Why this capability matters in real work."
  starting_point:
    provided:
      - "What the lab supplies or has already configured"
    learner_changes:
      - "The files, configuration, or resources the learner will change"
    not_required:
      - "Things the learner does not need to build, install, or provide"
  journey:
    - title: "Step one"
      description: "What the learner will do in this step."
      purpose: "Why this step matters."
    - title: "Step two"
      description: "What the learner will do in this step."
      purpose: "Why this step matters."
  success:
    criteria:
      - "An observable result that demonstrates success."
    final_state: "What is true when the learner finishes."
  tools:
    - "Terminal"
    - "Web IDE"
  important_notes:
    - "Temporary, destructive, or intentionally surprising behavior to know upfront."
steps:
  - title: "Step one"
    text: steps/step1.md
    verify: step1.sh
  - title: "Step two"
    text: steps/step2.md
```

`orientation` defines the repeatable first-time briefing shown before a lab. The UI always presents five pages: **mission**, **starting point**, **journey**, **how the lab works**, and **success and environment**. Authors provide scenario-specific content; detailed instructions remain in the step Markdown files.

The following orientation fields are required: `mission`, `starting_point.provided`, at least one `starting_point.learner_changes` or `starting_point.not_required`, one `journey` entry for every step, `success.criteria`, `success.final_state`, and at least one `tools` entry. Each journey entry requires `title`, `description`, and `purpose`. Scenario loading rejects incomplete orientations so new scenarios cannot silently ship without a first-time briefing.

Keep orientation content concise enough to scan across the five pages. Explain the starting point, learner responsibility, and observable finish line; do not duplicate the step instructions. `why`, `important_notes`, and `learner_changes` or `not_required` may be empty when they do not apply; `provided` must identify at least one starting-point item. Playgrounds use the same contract, with `learner_changes: []` and a single journey entry for their welcome step.

The catalog order, sections, and external learning resources are defined centrally in `scenarios/catalog.yaml`. Add a `scenario: <directory-id>` item there to make a scenario visible in the learner catalog. Prerequisites are advisory and never block launch.

The `tier` field determines which golden PVC is cloned for the session:
- `base` — minimal Ubuntu + terminal tools
- `tools` — base + Docker, k3d, uds CLI
- `uds-core` — tools + a running k3d UDS Core cluster (ready immediately)

**Services** (`services:`) declares named URLs shown as clickable chips in the terminal header:

```yaml
services:
  - label: "SSO (Keycloak)"
    url: "https://sso.uds.dev"
  - label: "Grafana"
    url: "https://grafana.admin.uds.dev"
```

### Step Markdown code blocks

Only fenced shell blocks are injected into the lab terminal when clicked. Label
commands with `bash` (or `sh`, `shell`, or `zsh`):

````markdown
```bash
kubectl get pods -A
```
````

Other fenced blocks, including YAML configuration and “wrong way” examples,
are display-only and retain syntax highlighting. Unlabeled fences are also
display-only, so the run behavior is explicit for every command.

### setup.sh

Runs in the background on the VM after boot. Must touch `/var/log/lab-setup/ready` when complete.

```bash
#!/bin/bash
set -euo pipefail
export HOME=/root

# scenario-specific setup...

touch /var/log/lab-setup/ready
```

For `uds-core` tier scenarios, the k3d cluster is stopped before snapshotting and
must be restarted in `setup.sh`:

```bash
systemctl start docker
k3d cluster start uds
k3d kubeconfig get uds > /root/.kube/config
touch /var/log/lab-setup/ready
```

### Verify scripts

`verify/step<N>.sh` — exit 0 = pass. Run as root on the VM, 30-second timeout.

```bash
#!/bin/bash
export HOME=/root
kubectl get ns my-namespace &>/dev/null
```

### DNS inside the VM

VMs running an inner k3d/k3s cluster need `*.uds.dev` to resolve to `127.0.0.1`
(the inner cluster's ingress), not the outer cluster's MetalLB IPs. This is handled
automatically by dnsmasq in `user-data.sh.gotmpl`:

```
address=/.uds.dev/127.0.0.1   # wildcard — inner cluster
server=1.1.1.1                  # internet DNS
server=8.8.8.8
```

## Development

### Project structure

```
cmd/
  labserver/    # HTTP server: sessions API, WebSocket proxy
  laboperator/  # Kubernetes operator: reconciles LabSession CRDs → VMIs
internal/
  operator/     # operator config, controller
  provider/
    kubevirt/   # KubeVirt provider: VMI + DataVolume + Service + NetworkPolicy
  session/      # session manager, session state
packer/         # QEMU packer builds for each VM tier
chart/          # Helm chart for uds-labs deployment
tasks/          # grouped UDS development tasks
vm/             # user-data.sh.gotmpl — cloud-init for lab VMs
scenarios/      # lab scenario definitions

~/src/github.com/uds-packages/
  containerized-data-importer/ # External CDI package source and artifacts
```

### Deployment configuration

The default deployment bundle consumes the published KubeVirt and CDI UDS packages. Generic
bundle deployments use the ignored `bundle/uds-config.yaml`, copied from
`bundle/uds-config.example.yaml`. The local E2E and redeploy workflows use the tracked portable
`bundle/uds-config-local.yaml` by default; override them with `--with BUNDLE_CONFIG=...` when needed.
Other deployment tasks use the ignored environment-specific config by default. The canonical bundle
uses published infrastructure packages and deploy-time configuration.
Package tarballs and environment configuration remain ignored build artifacts.

### Iterating on the operator

```bash
# Make code changes, then:
uds run redeploy

# Watch operator logs
kubectl logs -n uds-labs -l app=lab-operator -f

# Create a test session with an expiry relative to the current time
uds run create-test-session
kubectl get labsession -A -w
kubectl get vmi -n uds-labs-vms -w
```

### Session Management

Each browser is identified by a `lab_client_id` cookie (HttpOnly, 30-day expiry). Only one active lab session is allowed per client — attempting to start a second returns HTTP 409. The existing session can be ended from the lab UI or by waiting for the TTL to expire.
