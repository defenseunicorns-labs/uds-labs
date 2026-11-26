# Educates as an arm64 replacement for the UDS Labs VM platform

**Investigated:** 2026-11-26  
**UDS Labs revision:** `6f69ce3eefb1a1a319f88731a4da44d1af4629e9`  
**Educates stable release:** `3.8.0` (`479c9a172a4a4bbafaaf95a9d4206a31d61ea426`)  
**Related assessment:** [Educates fit assessment](2026-10-01-educates-fit-assessment.md)

## Recommendation

Proceed with an **arm64 proof of concept**. Educates is a credible replacement for KubeVirt/CDI, but only if the project adopts Educates as the workshop platform rather than treating it as another implementation of the existing `LabSession` provider.

The preferred target is:

```text
arm64 Kubernetes cluster
├── Educates: catalog, dashboard, terminal/editor, lifecycle, quotas, grading
├── cluster-native workshop Pods and per-session namespaces
└── one shared UDS Core installation for deployment-oriented exercises
```

This removes KubeVirt, CDI, QEMU, golden PVC cloning, Packer, VM image servers, the Labs operator, the custom session proxy, noVNC, and the nested VM boundary. It is substantially lighter per session, but it changes the current product model: learners work in containers and namespaces against a shared UDS Core installation rather than receiving a complete mutable VM containing a private, pre-baked UDS Core cluster.

Do not begin by porting the UDS Core Playground. First port **Package a Stateless App for UDS** and prove arm64, routing, Docker builds, isolation, and cleanup.

## Arm64 finding

**Educates itself supports arm64.** This is documented support, not merely an inference from source portability:

- Educates added native arm64 deployment support in 2.1.0. Source: [2.1.0 release notes](https://github.com/educates/educates-training-platform/blob/3.8.0/project-docs/release-notes/version-2.1.0.md).
- Release 3.8.0 publishes Linux and macOS arm64 CLI binaries. Source: [3.8.0 release assets](https://github.com/educates/educates-training-platform/releases/tag/3.8.0).
- The 3.8.0 release workflow builds platform and workshop images for `linux/amd64,linux/arm64`. Source: [3.8.0 image workflow](https://github.com/educates/educates-training-platform/blob/3.8.0/.github/workflows/build-and-publish-images.yaml).
- Educates exposes `platform_arch` as `amd64` or `arm64` to workshop definitions and content. Source: [Workshop definition](https://github.com/educates/educates-training-platform/blob/3.8.0/project-docs/custom-resources/workshop-definition.md).

Registry manifests were also checked directly for 3.8.0. The core `base-environment`, `session-manager`, `training-portal`, `secrets-manager`, `tunnel-manager`, `image-cache`, `assets-server`, `lookup-service`, `node-ca-injector`, CLI, Docker extension, and published JDK/Conda workshop images all contain `linux/arm64` variants. The pinned optional/runtime dependencies checked—including Docker-in-Docker, Contour, Envoy, cert-manager, Kyverno, kapp-controller, vcluster, and its Kubernetes images—also publish arm64 variants.

The arm64 requirement therefore does **not** require emulation and does not require KubeVirt.

### UDS toolchain support

The currently pinned lab toolchain also has arm64 artifacts:

- UDS CLI `v0.35.1` publishes `Linux_arm64`, arm64 DEB, and arm64 RPM assets. Source: [UDS CLI v0.35.1 release](https://github.com/defenseunicorns/uds-cli/releases/tag/v0.35.1).
- Zarf `v0.83.0` publishes a Linux arm64 binary and arm64 init package. Source: [Zarf v0.83.0 release](https://github.com/zarf-dev/zarf/releases/tag/v0.83.0).
- UDS Core's release pipeline builds and publishes amd64 and arm64 packages and k3d bundles. Source: [UDS Core publish tasks](https://github.com/defenseunicorns/uds-core/blob/2d26151b4a7a425977af28e312a853efcb3e783c/tasks/publish.yaml).

This does not prove that every third-party image used by every future workshop is multi-architecture. Workshop images, example application images, extension packages, and any UDS dependencies must still pass an arm64 manifest audit.

## Why this is lighter

The current medium Lab Session requests **4 CPU, 8 GiB RAM, and a 40 GiB cloned disk**, before accounting for KubeVirt/CDI platform overhead. It then runs QEMU, a guest OS, Docker, k3d, and UDS Core inside the VM. Sources: [`internal/sizing/sizing.go`](../../internal/sizing/sizing.go) and [`chart/values.yaml`](../../chart/values.yaml).

Educates 3.8 defaults are much smaller:

- workshop container: 512 MiB, or 1 GiB with the editor enabled;
- optional Docker sidecar: 768 MiB and a 5 GiB PVC;
- optional workshop home storage: explicitly sized only when required.

Sources: [workshop memory guidance](https://github.com/educates/educates-training-platform/blob/3.8.0/project-docs/production-guides/review-guidelines/workshop-container-memory.md), [Docker resource guidance](https://github.com/educates/educates-training-platform/blob/3.8.0/project-docs/production-guides/review-guidelines/docker-resource-requirements.md), and [storage guidance](https://github.com/educates/educates-training-platform/blob/3.8.0/project-docs/production-guides/review-guidelines/workshop-container-storage.md).

The 3.8.0 arm64 workshop base image is about 1.17 GiB compressed. That image is cached on nodes and shared by sessions instead of allocating a 40 GiB PVC clone per learner.

These numbers are not an expected production sizing for the UDS authoring labs. Builds may need 2–4 GiB in the workshop container plus Docker memory. They show the important architectural difference: session cost is Pod-sized and UDS Core can be amortized across sessions instead of duplicated inside every VM.

Educates is not itself a tiny production platform. Its official starting guidance for a hosted environment is three workers with 16–32 GiB each, depending on workshop count and requirements. A single-user arm64 installation can likely be much smaller, but that must be measured rather than treated as supported production sizing. Source: [cluster requirements](https://github.com/educates/educates-training-platform/blob/3.8.0/project-docs/installation-guides/cluster-requirements.md).

## Execution models

### 1. Shared host-cluster UDS Core — recommended

Install one UDS Core instance on the same dedicated cluster and give every Educates session a namespace, scoped credentials, quotas, and unique hostnames.

**Advantages**

- Lowest per-session cost and fastest startup.
- Closest to how an application team should consume a real shared UDS platform.
- Removes nested Kubernetes and the complete VM/image pipeline.
- Natural arm64 deployment: Educates, UDS CLI, Zarf, and UDS Core all publish arm64 artifacts.

**Costs**

- Current scenarios assume a private cluster and fixed `*.uds.dev` hosts. They must use session-unique namespaces, names, and hosts.
- Cluster-scoped package/operator installation cannot safely be repeated by every learner. Platform dependencies must be preinstalled, shared, or moved out of the exercise.
- Destructive or cluster-admin lessons need a different environment.
- The old VM isolation contract no longer applies; the threat model and [`docs/ux-and-security-contract.md`](../ux-and-security-contract.md) must be revised.

This is the correct model if “lightweight” is the primary goal.

### 2. k3d inside Educates Docker support — compatibility fallback

Educates can provide a privileged Docker-in-Docker sidecar. A workshop container could run k3d and deploy a private UDS Core cluster into that daemon, preserving much of the current guest-local topology without a VM.

**Advantages**

- Preserves a private cluster, cluster-admin exercises, fixed internal hostnames, and much of the existing setup content.
- Eliminates KubeVirt, CDI, Packer, qcow2 images, and 40 GiB PVC clones.
- Docker-in-Docker and the pinned UDS tools have arm64 builds.

**Costs**

- Still duplicates UDS Core per session, so compute savings are limited.
- Educates documents Docker support as privileged and cluster-compromisable; use only on a disposable dedicated cluster.
- The current golden image starts from a pre-baked UDS Core installation. Rebuilding UDS Core at session start may make startup unacceptable unless a new cache/preload strategy is proven.
- This is still nested Kubernetes, just in a Pod rather than a VM.

Use this only for scenarios that cannot be redesigned for a shared platform.

### 3. Per-session vcluster — defer

Educates can create a vcluster for each session and the shipped vcluster images support arm64. It may provide a cluster-admin-shaped experience with less overhead than a VM.

Do not choose it for the first spike. UDS Core uses CRDs, admission behavior, operators, gateways, network policy, and cluster-scoped resources. Their behavior through vcluster synchronization and routing needs dedicated validation. It is neither a drop-in replacement for the current nested k3d topology nor clearly lighter after installing UDS Core per session.

## Educates is a platform replacement, not a provider implementation

The existing UDS Labs provider seam creates infrastructure behind a custom catalog, server, proxy, injection service, verifier, and UI. Educates already owns all of those concerns through `Workshop`, `TrainingPortal`, `WorkshopSession`, its dashboard, terminal/editor, ingress handling, and examiner actions.

Trying to keep the current UDS Labs UI while using Educates only to create Pods would retain most bespoke code and add a second lifecycle model. The cleaner migration is:

| Current UDS Labs concern | Educates replacement |
|---|---|
| Scenario catalog and metadata | `Workshop` + `TrainingPortal` |
| LabSession/operator lifecycle | `WorkshopSession` and session manager |
| ttyd/injection proxy | Educates terminal and clickable actions |
| Browser/noVNC | learner browser + session/application ingress |
| `verify/*.sh` | examiner actions/tests |
| VM size/image fields | session resources, applications, packages, namespace budget |
| VM setup script | workshop setup scripts and declarative session objects |
| KubeVirt/CDI/golden PVCs | workshop Pod, namespace, optional PVC/Docker/vcluster |

The content can be migrated incrementally, but the target should retire the custom control plane rather than place Educates beneath it.

## Main integration risks

### Ingress

Educates 3.8 creates Kubernetes `Ingress` resources and recommends Contour. UDS Core normally exposes applications through Istio resources generated from UDS `Package` resources. There is no documented Educates-to-UDS ingress integration.

The lowest-risk proof of concept is to run a separate Contour ingress class and workshop domain for Educates, while Istio continues to own UDS application hosts. On a bare-metal or single-node cluster this usually requires separate load-balancer addresses so both stacks can serve 80/443. A later integration could translate Educates routes to Istio, but that is not required to prove the workshop model.

### Policy engines

Educates relies on Kyverno for workshop isolation. UDS Core uses Pepr policies. Assume both are required for the first untrusted-user deployment and test their admission interactions. Do not treat Falco as a replacement for Educates' admission policies.

### Per-session UDS identities and hosts

The current scenarios use fixed namespaces and fixed service names. A shared UDS Core model needs every generated resource to incorporate Educates variables such as the session name. Application `Package` resources need unique hosts, and browser links need to be generated from those hosts.

### Docker and image architecture

An arm64 Docker daemon builds arm64 images by default. That is correct when the target cluster is arm64, but workshop instructions and verifiers must stop assuming `amd64` package names. If a lesson is intended to produce portable artifacts, it must explicitly teach/build multi-architecture images and create the corresponding Zarf/UDS artifacts.

### Air gap

Educates supports relocatable OCI workshop content and configurable registries, but the complete 3.8 Carvel installer image set and workshop dependencies still need to be packaged into the UDS/Zarf delivery path. Do not make air-gap packaging part of the first product-fit spike; inventory it immediately after the runtime path works.

### Stable 3.8 versus upcoming 4.0

The latest stable release is 3.8.0 and uses the Carvel-based installation path. The upstream main branch is developing a Helm/operator-based 4.0 installation. Use stable 3.8.0 for the proof of concept, but avoid investing heavily in custom 3.x installer integration that will be discarded during a 4.0 upgrade.

## Scenario disposition

| Current scenario | Educates direction |
|---|---|
| Tools Playground | Direct container workshop; no VM or UDS Core required. |
| Package a Stateless App for UDS | First migration candidate. Use arm64 workshop image, Docker sidecar only if the image must be built, shared UDS Core namespace, and a unique application host. |
| Package a Stateless Evidence App | Port after the first scenario. Preinstall shared operators; issue session-scoped database/object-store credentials and namespaces. |
| Explore UDS Package Structure | Port content/editor/examiner directly; deploy against shared Core or make deployment optional. |
| UDS Core Playground | Redefine as access to a dedicated/shared UDS Core installation, or use the Docker/k3d fallback. It cannot retain the exact private golden-VM experience while also removing the VM model. |

## Proposed arm64 spike

Use a disposable, native arm64 cluster. A single-node k3s cluster is acceptable for the spike; the production topology remains a separate decision.

1. Install Educates 3.8.0 without any amd64 emulation.
2. Publish a minimal multi-architecture UDS workshop image containing the pinned UDS CLI, Zarf, Helm, `kubectl`, and scenario files.
3. Run a terminal/editor-only workshop and record platform idle cost and session startup time.
4. Install a shared arm64 UDS Core instance and the minimum dependencies needed by **Package a Stateless App for UDS**.
5. Port that scenario with a per-session namespace and unique host.
6. Enable the arm64 Docker sidecar only for the image-build step; deploy the resulting arm64 image to the shared cluster.
7. Run two concurrent sessions and test namespace, credential, network, host, and cleanup isolation.
8. Delete a session and verify all session namespaces, credentials, routes, PVCs, and workload objects are removed.

### Go/no-go criteria

- Every running image resolves to `linux/arm64`; no QEMU/binfmt emulation is installed.
- A session is usable in under 60 seconds before scenario-specific setup.
- The full stateless scenario starts materially faster than the current cloned-VM path.
- Per-session reserved memory is materially below the current 8 GiB medium VM.
- Terminal websockets and application routes remain stable through session create/delete churn.
- Two sessions cannot read or mutate each other's namespaced resources or credentials.
- UDS Package reconciliation, generated network policy, and the application route work with session-unique names.
- Session expiry deterministically cleans up all owned resources.

## Decision

**Go forward with the spike.** Arm64 is supported throughout the Educates platform and the pinned UDS authoring toolchain. The key question is no longer CPU architecture; it is whether the learning product accepts a shared, cluster-native UDS Core model.

If yes, Educates can replace most of this repository's bespoke runtime and produce a large reduction in per-session compute, storage, and maintenance. If every learner must continue receiving a private, pre-baked UDS Core environment, Educates can still remove KubeVirt/CDI by using Docker/k3d, but the result will be less lightweight and will trade fast golden-image startup for privileged nested-container complexity.
