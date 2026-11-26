# KubePrep ideas for UDS Labs: existing-cluster assessment

**Investigated:** 2026-09-13  
**UDS Labs revision:** `73a24ab528a8cd149f10c6a543162793fd459821`  
**KubePrep revision:** `f8e28803c2499798906e998d7c756f0ce68606eb`  
**Related UDS Labs prototype:** `811ec7e839f3d0768cbeb83b14f5812b9163d8d9`

## Conclusion

Yes, UDS Labs can incorporate the cluster-native ideas from KubePrep, but KubePrep is not currently a client for an arbitrary pre-existing Kubernetes cluster. Its `start` path creates or verifies a **dedicated, framework-owned kind cluster**, then reuses that cluster. The transferable idea is to run labs as ordinary Kubernetes workloads and grade observable cluster state instead of provisioning a VM per learner.

For UDS Labs, the best fit is a second, cluster-native backend for a subset of labs:

```text
Browser → UDS Labs server → per-session terminal Pod
                              │
                              └── existing UDS/Kubernetes cluster
```

The existing KubeVirt backend should remain for labs that need a full Linux guest, systemd, Docker-in-Docker, nested k3d, or a pre-baked UDS Core disk. This is a capability split, not a wholesale replacement.

## What KubePrep actually does

KubePrep's own README describes a split terminal backed by a dedicated `kind` cluster and says that it grades cluster state rather than typed commands ([README](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/README.md)). Its architecture document identifies four relevant boundaries: cluster lifecycle, curriculum, a state validator, and terminal UI ([architecture](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/docs/architecture.md)).

Important qualification: `EnsureCluster` invokes `kind create cluster` when the named cluster is absent. If a cluster with that name exists without KubePrep's ownership record, it refuses to adopt it ([lifecycle.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/cluster/lifecycle.go)). KubePrep therefore demonstrates **reuse of a dedicated cluster**, not safe multi-tenant use of an arbitrary shared cluster.

The most reusable patterns are:

1. **A real shell with scoped cluster credentials.** The terminal process receives a dedicated `KUBECONFIG`, namespace, and scenario metadata ([shell.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/terminal/shell.go)).
2. **Namespace-as-workspace.** A lab can declare a namespace; setup/reset composes a namespace and trusted manifests, and reset deletes the workspace namespace ([root.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/cli/root.go), [resources.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/curriculum/resources.go)).
3. **Declarative setup/reset.** Curriculum references Kubernetes YAML rather than host shell commands ([schema.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/curriculum/schema.go)).
4. **State-based grading.** Checks include object existence, field equality, Pod readiness, Deployment availability, container health, and node topology ([checks.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/validator/checks.go)). The grader does not need command history.
5. **Explicit safety boundary.** Community curriculum rejects cluster-scoped kinds, host paths, host namespaces, privileged containers, and related fields ([safety.go](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/internal/curriculum/safety.go)). Its security document explicitly says the Lab Shell is not a sandbox ([security.md](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/docs/security.md)).

## What already exists in UDS Labs

UDS Labs already has the durable orchestration seam needed for this direction:

- `LabSession` is the lifecycle object and survives server restarts ([API type](https://github.com/defenseunicorns-labs/uds-labs/blob/73a24ab528a8cd149f10c6a543162793fd459821/api/v1alpha1/labsession_types.go)).
- The server proxies terminal, shell, browser, command injection, and verification traffic through a session Service ([server](https://github.com/defenseunicorns-labs/uds-labs/blob/73a24ab528a8cd149f10c6a543162793fd459821/cmd/labserver/main.go)).
- The provider interface isolates infrastructure from session lifecycle ([provider interface](https://github.com/defenseunicorns-labs/uds-labs/blob/73a24ab528a8cd149f10c6a543162793fd459821/internal/provider/provider.go)).
- The production provider currently creates a VMI, CDI DataVolume, Service, and NetworkPolicy per session ([KubeVirt provider](https://github.com/defenseunicorns-labs/uds-labs/blob/73a24ab528a8cd149f10c6a543162793fd459821/internal/provider/kubevirt/provider.go)).
- Scenario content already has setup scripts, verification scripts, browser mode, image tiers, and a scenario-specific workspace inside the VM ([scenario model](https://github.com/defenseunicorns-labs/uds-labs/blob/73a24ab528a8cd149f10c6a543162793fd459821/internal/scenario/scenario.go)).

There is also a prior UDS Labs branch containing an experimental plain-Pod provider. Commit [`811ec7e`](https://github.com/defenseunicorns-labs/uds-labs/commit/811ec7e839f3d0768cbeb83b14f5812b9163d8d9) creates a lab Pod, Service, and NetworkPolicy, adds `PROVIDER_TYPE=pod`, and documents use with k3d/OrbStack. It is not on the current `master` revision and its own notes restrict it to scenarios that do not require VM snapshots, systemd, Docker, or nested k3d ([provider](https://github.com/defenseunicorns-labs/uds-labs/blob/811ec7e839f3d0768cbeb83b14f5812b9163d8d9/internal/provider/pod/provider.go), [dev guide](https://github.com/defenseunicorns-labs/uds-labs/blob/811ec7e839f3d0768cbeb83b14f5812b9163d8/docs/dev-mac.md)). This is useful prior art, but it needs to be forward-ported and hardened rather than copied unchanged.

## Recommended architecture

### 1. Keep the current KubeVirt provider

Use it for the VM-shaped contract:

- systemd and a mutable Linux filesystem;
- Docker or another container runtime inside the lab;
- nested k3d and a pre-baked UDS Core cluster;
- Browser Mode that depends on guest-local DNS/loopback;
- scenarios whose setup assumes the `uds-core` golden disk.

The current scenario set heavily uses `image: uds-core` and `browser: true`; those fields are signals that these scenarios are not automatically portable to a plain Pod.

### 2. Add a cluster-native provider or mode

Forward-port the Pod provider, but treat it as a cluster-native session backend rather than a VM replacement. A session would create:

- a dedicated namespace such as `uds-lab-<session>`;
- a terminal/runtime Pod with ttyd, the injection service, and optional browser tooling;
- a Service for the existing UDS Labs proxy;
- a per-session ServiceAccount, Role/RoleBinding, ResourceQuota, LimitRange, and default-deny NetworkPolicies;
- a setup Job or init container that applies trusted starting state and signals readiness.

The existing `LabSession`, operator reconciliation, server proxy, TTL, and completed-step persistence can remain. The provider changes the child resources and readiness checks.

A per-session namespace is preferable to one shared `uds-labs-vms` namespace for this mode because namespace deletion gives reset/expiry a clear cleanup boundary. If operational constraints require one namespace, every resource and policy must still be selected by an unguessable session label and cleanup must be comprehensive.

### 3. Add a KubePrep-style scenario contract

For cluster-native scenarios, add a declarative flavor that can express:

- required Kubernetes version/capabilities;
- trusted starting manifests;
- reset manifests or namespace reset behavior;
- typed observable checks;
- whether the lab is namespace-scoped or requires an isolated cluster.

The current shell-based `setup.sh`/`verify/*.sh` contract can remain for VM labs. It should not be assumed to be safe or portable when executed against a shared cluster.

The first cluster-native scenarios should use checks equivalent to KubePrep's object/field/readiness checks. This makes completion independent of command syntax and avoids storing shell history. Existing UDS Labs step verification can continue for VM labs.

### 4. Give the runtime Pod deliberately scoped Kubernetes credentials

A Pod does not automatically give a learner a usable `kubectl` context. The backend needs to create a ServiceAccount and either:

- generate a kubeconfig for that ServiceAccount in an init step; or
- mount projected ServiceAccount credentials and configure `kubectl`/the UDS CLI to use them.

The Role must be the minimum needed by the scenario. Read-only cluster-wide discovery can be a separate, explicit permission. A shared-cluster learner must not receive unrestricted cluster-admin credentials merely because a scenario runs `kubectl`.

This is a major difference from local KubePrep: KubePrep can point the learner's host shell at its dedicated kubeconfig, while UDS Labs must put credentials inside a browser-reachable runtime environment.

### 5. Never destroy an externally supplied cluster

For an existing-cluster mode, cluster lifecycle is outside UDS Labs. The platform may create and delete only its namespaces and owned resources. It should require an explicit deployment mode and refuse cluster-scoped setup unless the cluster is marked dedicated to the workshop.

The KubePrep ownership checks are a useful model for local cluster management, but for a shared/external cluster the safer rule is stronger: no adoption or destruction at all, and no mutation outside an allowlisted scope.

## Compatibility

| Lab capability | Existing-cluster / Pod mode | Reason |
|---|---:|---|
| Basic Pods, Deployments, Services, labels, readiness, field inspection | **Yes** | These are ordinary Kubernetes API/state exercises. |
| KubePrep-style declarative setup/reset and state grading | **Yes** | A per-session namespace plus typed checks maps cleanly. |
| UDS package authoring against an already-running UDS Core cluster | **Possible** | The runtime needs `uds`/Zarf, registry access, correct UDS Core permissions, and namespace/bundle conventions. |
| Existing cluster's shared Istio/UDS services | **Possible** | Requires explicit namespace, host, ingress, and policy design; it is not guest-loopback DNS. |
| Browser/noVNC | **Possible, not automatic** | The runtime image must include Chromium and VNC startup; DNS/ingress behavior must be redesigned for the host cluster. |
| `docker build` inside a lab | **No, by default** | An ordinary Pod has no Docker daemon and should not be made privileged on a shared cluster. Use prebuilt images or a controlled build service. |
| systemd, full mutable OS, SSH, VM-level networking | **No** | These are VM capabilities. |
| Nested k3d / pre-baked `uds-core` disk | **No, as-is** | Remove the nested topology from the scenario or retain KubeVirt. |
| Node/cluster-admin/CRD/webhook exercises | **Only on an isolated cluster** | Namespace isolation cannot safely contain cluster-scoped mutations. |

## Main risks and mitigations

1. **Cross-learner contamination.** Use per-session namespaces, unique names, quotas, default-deny policies, and deletion on expiry. Never share mutable initial state without a reset boundary.
2. **Credential escalation.** Use per-session ServiceAccounts and Roles. Do not mount the platform server's credentials into learner Pods. Reject or separately classify cluster-scoped scenarios.
3. **Host compromise through the lab shell.** Treat the runtime as untrusted, following KubePrep's own warning that its shell is not a sandbox. Use a disposable, dedicated cluster for any scenario granting broad permissions; enforce Pod Security and avoid host mounts/privileged containers.
4. **Image and registry assumptions.** The existing-cluster mode needs a cluster-local image strategy for air-gapped deployments and must not depend on Docker socket access.
5. **Browser and service routing.** VM Browser Mode currently relies on guest-local `*.uds.dev` behavior. Host-cluster mode needs host-cluster DNS and ingress semantics, plus a decision about whether users share platform services or get per-session copies.
6. **Setup lifecycle.** Init containers share only explicitly mounted volumes; files written to an init container's root filesystem are not visible to the main container. Scenario setup artifacts must use a shared volume or be produced by the runtime container.
7. **Capacity and noisy neighbors.** ResourceQuota/LimitRange and platform-level admission policy are required even if the operator also sets Pod requests.

Kubernetes' namespace, multi-tenancy, NetworkPolicy, ServiceAccount/RBAC, ResourceQuota, and Pod Security documentation should be treated as the baseline for the implementation:

- [Namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/)
- [Multi-tenancy](https://kubernetes.io/docs/concepts/security/multi-tenancy/)
- [Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Service Accounts](https://kubernetes.io/docs/concepts/security/service-accounts/)
- [Using RBAC Authorization](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
- [Resource Quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/)
- [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)

## Suggested implementation order

1. Define a `cluster-native` scenario capability and classify existing scenarios; do not silently run VM scenarios on Pods.
2. Forward-port the existing experimental Pod provider against current `master`, but first make the provider contract explicit about namespace, ServiceAccount, setup, browser support, and pause behavior.
3. Add a per-session namespace and scoped credentials; test two simultaneous sessions and cross-session access.
4. Port one small, namespace-scoped Kubernetes lab with declarative setup/reset and typed state checks.
5. Add deployment configuration selecting `kubevirt` or `pod`/`cluster-native`, defaulting to KubeVirt for backward compatibility.
6. Validate the cluster-native mode on the actual target UDS Core cluster, including DNS, Istio, NetworkPolicy enforcement, image pulls, quota behavior, expiry cleanup, and operator restart recovery.
7. Only then port UDS packaging labs that do not require Docker-in-Docker or nested k3d.

## Decision recommendation

Proceed with a **dual backend**:

- **KubeVirt** for the existing VM-centric, air-gapped, nested-UDS-Core experience.
- **Cluster-native Pod sessions** for Kubernetes/UDS labs that can run against an existing, dedicated or properly isolated UDS Core cluster.

Borrow KubePrep's scenario schema, namespace reset, scoped kubeconfig, typed state validator, and safety review. Do not adopt its dedicated-kind lifecycle literally, and do not remove KubeVirt until the VM-dependent scenarios have been redesigned or explicitly retained.

KubePrep is Apache-2.0 licensed ([LICENSE](https://github.com/YakShavingCatHerder/kubeprep/blob/f8e28803c2499798906e998d7c756f0ce68606eb/LICENSE)); direct code reuse should preserve the required notices and be reviewed separately from adopting the architectural ideas.
