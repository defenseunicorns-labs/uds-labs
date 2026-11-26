# Educates fit assessment for UDS Labs

**Investigated:** 2026-10-01  
**UDS Labs revision:** `6f69ce3eefb1a1a319f88731a4da44d1af4629e9`  
**Educates source revision:** `b026a103aa448bb8367a5bf47d9e4079475566b3`  
**Question:** Is Educates a better fit for the workshop scenarios UDS Labs is building, especially given Educates' configurable workshop environments and multiple content configurations?

## Conclusion

Educates is a better fit for the **workshop platform and learner experience** than the current bespoke UDS Labs UI/runtime for most of the planned scenarios. It already provides the configurable workshop model UDS Labs is moving toward: a workshop can select a container, Kubernetes namespace, vcluster, KubeVirt VM, or remote VM; these can be combined; and each session receives scoped credentials, quotas, storage, ingress, and optional applications such as an editor, Kubernetes console, Docker daemon, image registry, and test examiner.

It is **not a drop-in replacement for the current VM runtime**. UDS Labs currently depends on golden PVC clones, a mutable Linux VM, systemd, Docker, nested k3d, a pre-baked UDS Core cluster, guest-local DNS, and an embedded Chromium/noVNC browser. Educates can provision KubeVirt resources, but its documented model does not make this golden-image/nested-UDS-Core workflow a first-class abstraction.

There is an important deployment nuance: Educates can target an existing cluster, but its current Inline/ExternalIngressController integration expects a Kubernetes `IngressClass` and generates Kubernetes `Ingress` resources. UDS Core's documented ingress model is Istio gateways plus UDS `Package` resources that generate Istio `VirtualService` and `AuthorizationPolicy` objects. Educates does not document a UDS Core/Istio gateway provider. This is an **implementation seam, not a fundamental architectural blocker**: Educates could be adapted to emit Istio/UDS routing resources, or a bridge controller could translate Educates' generated `Ingress` objects into `VirtualService` objects.

Recommended direction:

```text
Educates = workshop catalog, dashboard, session lifecycle, content, configuration, and grading
KubeVirt/UDS Labs VM capability = retained for VM-shaped scenarios
Cluster-native Educates sessions = preferred target for ordinary UDS/Kubernetes scenarios
```

Do not port every scenario immediately. First build one Educates-native version of **Package a Stateless App for UDS** against a shared/dedicated UDS Core cluster, then test whether **Release Ledger** works with the required operator and bundle boundaries.

## What Educates provides that maps well

### Environment composition

Educates supports these session environment choices:

- container only;
- a dedicated Kubernetes namespace with RBAC and resource quotas;
- a per-session vcluster with cluster-admin access scoped to the virtual cluster;
- a KubeVirt VM;
- a remote VM through infrastructure operators such as Crossplane.

The choices can be combined in a single workshop. This directly addresses the need to express different workshop profiles without adding more bespoke runtime code. Sources: [Educates project overview](https://docs.educates.dev/en/stable/project-details/project-overview.html) and [platform architecture](https://docs.educates.dev/en/stable/project-details/platform-architecture.html).

### Content and content variations

Workshop instructions are Markdown and can be distributed from Git, an OCI artifact, or HTTP. A workshop can use a standard or custom image, extension packages, setup scripts, environment variables, and pre-created Kubernetes resources.

Educates has two useful variation mechanisms:

- the classic renderer can activate different module sets using different workshop configuration files and `WORKSHOP_FILE`;
- the Hugo renderer can define multiple navigation pathways and select one with `PATHWAY_NAME`.

This is a stronger native model for workshop variants than adding more fields to the current `scenario.yaml`. Sources: [platform architecture](https://docs.educates.dev/en/stable/project-details/platform-architecture.html), [workshop configuration](https://docs.educates.dev/en/stable/workshop-content/workshop-configuration.html), and [Workshop Definition](https://docs.educates.dev/en/stable/custom-resources/workshop-definition.html).

### Learner interface

Educates includes the integrated instructions panel, multiple terminals, a VS Code editor, a Kubernetes console, slides, and custom dashboard tabs. It supports clickable terminal commands and copyable text. This covers most of the current UDS Labs dashboard goals without maintaining a custom web UI.

Sources: [project overview](https://docs.educates.dev/en/stable/project-details/project-overview.html) and [platform architecture](https://docs.educates.dev/en/stable/project-details/platform-architecture.html).

### Session lifecycle and operations

The TrainingPortal manages workshop catalogs, reserved/on-demand capacity, session expiration, orphan detection, session extension, registration, and REST API access. It can also report active sessions and emit analytics webhooks.

This is broader than the current UDS Labs lifecycle, which has a useful `LabSession` abstraction but still has alpha limitations around server-restart auth state, session resume, hard TTLs, and orphaned sessions. Source: [TrainingPortal documentation](https://docs.educates.dev/en/stable/custom-resources/training-portal.html) and local UDS Labs domain notes in [`CONTEXT.md`](../../CONTEXT.md).

### Verification

The Educates examiner provides clickable or automatically-triggered tests backed by executable scripts, with arguments, timeout, retries, delay, forms, and optional backend URLs. This maps well to the planned static, platform, and behavioral checks for the Postgres/MinIO scenarios and the repair scenario.

The current UDS Labs `verify/step<N>.sh` contract is simpler and has a stronger built-in notion of blocking progression. Educates' documented primitive is a test action; it should be validated in a prototype before assuming it can enforce “must pass before Continue” exactly as the current UI does.

Sources: [examiner actions documentation](https://docs.educates.dev/en/stable/workshop-content/workshop-instructions.html#clickable-actions-for-the-examiner) and the official [examiner sample](https://github.com/educates/educates-training-platform/tree/b026a103aa448bb8367a5bf47d9e4079475566b3/workshop-samples/lab-examiner-actions).

## Fit against the current scenarios

| Scenario | Fit | Assessment |
|---|---:|---|
| Tools Playground | **High** | A container-only workshop is a direct fit. Use a custom image or extension packages for the tools. |
| Package a Stateless App for UDS | **High after redesign** | Use a custom image containing `uds`/Zarf, enable the per-session Docker daemon if learners build images, and use a session namespace or vcluster. Replace guest-local browser/noVNC with Educates ingresses or dashboard tabs. |
| Explore UDS Package Structure | **Medium** | The terminal/editor/examiner experience fits. The current pre-baked UDS Core + Postgres workflow needs either a shared dedicated UDS Core cluster, carefully preinstalled dependencies, or a vcluster design. |
| Package a Stateless Evidence App / Release Ledger | **Medium–High** | Educates can provide a namespace, storage, objects, secrets, and tests. The main design question is where Postgres/MinIO operators live and whether learners should install them per session, use shared operators, or receive a vcluster. |
| UDS Core Playground | **Low as a direct port; possible as a redesign** | A vcluster or KubeVirt VM may host a UDS Core exercise, but the current nested k3d/pre-baked golden PVC approach is not the natural Educates path. Keep this as a VM-shaped scenario until proven otherwise. |

The classification is based on the actual UDS Labs scenario definitions: the package-authoring scenarios use `browser: true` and `image: uds-core`, while the playgrounds also use `browser: true`; see [`scenarios/*/scenario.yaml`](../../scenarios/).

## Where Educates is better

1. **Configuration is a platform feature.** Resource budgets, RBAC roles, Kubernetes access, vcluster, Docker, registry, editor, console, ingress, storage, custom objects, and session parameters are declarative workshop settings.
2. **The content model is designed for variants.** Pathways/modules let one content repository produce multiple workshop shapes without duplicating all instructions.
3. **The default UX is already complete.** Terminal, editor, browser routing, instructions, files, slides, and custom tabs are built in.
4. **Session operations are mature.** Reserved pools, capacity controls, expiration, extension, orphan handling, portal auth, REST APIs, and analytics are already modeled.
5. **Grading is closer to the scenario roadmap.** Examiner checks can run against the session environment and can be retried while a deployment becomes ready.
6. **It aligns with the cluster-native direction already identified in this repository.** The existing KubePrep assessment recommends a dual backend: retain KubeVirt for VM-dependent labs and add cluster-native sessions for ordinary Kubernetes labs. Educates already supplies most of the cluster-native session machinery. See [`docs/research/2026-09-13-kubeprep-existing-cluster-assessment.md`](2026-09-13-kubeprep-existing-cluster-assessment.md).

## Where UDS Labs is currently better

1. **Golden VM startup and reproducibility.** UDS Labs clones golden PVCs containing a complete mutable Linux environment and, for `uds-core`, a pre-baked inner k3d/UDS Core installation. Educates' normal environment is a workshop container plus optional Kubernetes resources.
2. **Guest-local browser semantics.** UDS Labs intentionally resolves `*.uds.dev` to loopback inside the VM and runs Chromium/noVNC there. Educates normally exposes applications through cluster ingress and the learner's browser; this is a different networking model.
3. **Air-gapped artifact workflow.** UDS Labs is designed around Zarf bundles and packaged image servers, with no external registry address in DataVolumes. Educates supports OCI-hosted workshop content and images, but the complete UDS Labs air-gap workflow would need validation and likely custom packaging.
4. **Exact progression semantics.** UDS Labs has per-step verifier scripts and blocks the next step when a verifier is required. Educates examiner actions are powerful, but the same gating behavior needs to be implemented or accepted as a UX change.
5. **Existing UDS-specific operational behavior.** The current operator understands the LabSession → VMI/DataVolume/Service/NetworkPolicy lifecycle, pause/resume, and golden image tiers. Replacing that removes bespoke code, but also requires reproducing the behaviors that are important for UDS Core labs.

## Important constraints and risks

### Educates should run on a dedicated cluster

Educates explicitly recommends a dedicated Kubernetes cluster because workshop users may be able to affect cluster-wide behavior despite RBAC, quotas, and policy enforcement. It also requires a policy engine such as Kyverno for stronger workshop security enforcement. This matters for UDS Labs because the workshop scenarios intentionally teach packaging, cluster resources, Docker, and potentially privileged operations.

Source: [Educates cluster requirements](https://docs.educates.dev/en/stable/installation-guides/cluster-requirements.html).

### Docker support is useful but privileged

Educates can give each session a separate Docker daemon with persistent storage, which is attractive for the package-authoring scenarios. The official documentation warns that this requires a privileged container and recommends disposable clusters for untrusted users. That warning applies directly to `uds run dev` scenarios that build images.

Source: [Enabling Docker](https://docs.educates.dev/en/stable/custom-resources/workshop-definition.html#enabling-ability-to-use-docker) and [Docker resource requirements](https://docs.educates.dev/en/stable/production-guides/review-guidelines/docker-resource-requirements.html).

### UDS Core is not automatically solved

Educates provides the workshop environment and can connect it to a namespace, vcluster, or VM; it does not provide UDS Core as a built-in dependency. A viable deployment must choose one of these models:

- run Educates and all workshops on a dedicated UDS Core cluster **plus a compatible Kubernetes Ingress controller and Kyverno policy path for Educates**;
- add an Educates-specific Contour/Ingress entry point to the UDS Core cluster, using a separate workshop domain and TLS secret, then validate coexistence with Istio, Pepr, and UDS network/policy rules;
- extend Educates so its portal/session routes use UDS Core's Istio tenant/admin or a dedicated Istio gateway instead of Kubernetes `Ingress`;
- install the required UDS Core components into a per-session vcluster, if that topology is supported and affordable;
- provision a KubeVirt VM and retain the current UDS Labs-style prebuilt environment for the few scenarios that require it.

The first option is likely the best for package-authoring scenarios, but it changes the lesson from “start a private UDS Core VM” to “package against an existing UDS Core platform.” That is probably a better lesson for most ISV learners, but it is a product decision rather than a mechanical migration.

The security-policy detail is also significant. Educates' supported policy choices are Kyverno, Pod Security Standards, OpenShift SCC, or None. UDS Core provides **Pepr admission policies** and **Falco runtime detection**; Falco is not an admission/policy engine and does not replace Kyverno's workshop-isolation role. UDS Core therefore has strong security controls, but it does not satisfy Educates' documented workshop-level isolation contract through Falco alone. Running Kyverno alongside UDS Core may be viable, or Educates could be configured with workshop policy enforcement disabled for a trusted-user deployment, but either choice should be treated as a compatibility/security review rather than assumed coexistence. Sources: [Educates cluster requirements](https://docs.educates.dev/en/stable/installation-guides/cluster-requirements.html), [Educates configuration API](https://github.com/educates/educates-training-platform/blob/b026a103aa448bb8367a5bf47d9e4079475566b3/installer/operator/api/config/v1alpha1/educatesclusterconfig_types.go), [UDS Core policy engine](https://github.com/defenseunicorns/uds-core/blob/5c1153b9146c95cf9c2cce4366c72a8a0c9c199f/docs/reference/operator-and-crds/policy-engine.mdx), and [UDS Core runtime security](https://github.com/defenseunicorns/uds-core/blob/5c1153b9146c95cf9c2cce4366c72a8a0c9c199f/docs/concepts/platform/security.mdx).

### Authentication is configurable, not identical

Educates supports portal registration, anonymous access, a shared event password, and a REST API intended for a custom frontend with an external identity provider. This can implement the current workshop-code gate and GitHub OAuth through a thin external entry point, but those two steps are not the same as the current UDS Labs auth implementation out of the box.

Sources: [portal client authentication](https://docs.educates.dev/en/stable/portal-rest-api/client-authentication.html), [anonymous access](https://docs.educates.dev/en/stable/portal-rest-api/anonymous-access.html), and [TrainingPortal event access code](https://docs.educates.dev/en/stable/custom-resources/training-portal.html#specifying-an-event-access-code).

## Recommended migration shape

### Phase 1: prove the cluster-native authoring path

Create one Educates workshop for **Package a Stateless App for UDS** with:

- a custom workshop image containing the pinned UDS CLI, Zarf, Helm, Docker client, `kubectl`, and authoring tools;
- Educates Docker support only if the learner must build an image;
- a dedicated session namespace with narrowly scoped RBAC;
- a pre-existing dedicated UDS Core cluster or a documented shared platform namespace model;
- Educates `session.ingresses` for the packaged application's web endpoint;
- examiner tests for file state, package rendering, deployment readiness, and browser reachability;
- no nested k3d and no guest-local Chromium in the first experiment.

Measure startup time, per-session resource cost, package build behavior, UDS network policy behavior, and two simultaneous sessions.

### Phase 2: port Release Ledger

Use the same workshop image and add the operator-backed database/object-store path. Decide explicitly whether dependencies are:

- shared platform services with per-session credentials;
- installed into each session namespace by an already-installed operator; or
- installed inside a vcluster.

Keep the behavioral checks from the current plan: data survives application pod replacement, object checksums match, and deletion removes the object.

### Phase 3: retain VM scenarios selectively

Keep the current KubeVirt provider and golden PVCs for UDS Core Playground, nested-cluster exercises, systemd/Docker-in-a-guest exercises, and any scenario whose learning objective is specifically a complete Linux environment. These can later be exposed through Educates using `session.objects` and KubeVirt, but that integration should be treated as an experiment rather than assumed compatibility.

### Phase 4: move catalog and portal responsibility

If the first two phases work, stop expanding the custom UDS Labs catalog/session UI. Represent scenarios as Educates `Workshop` definitions, use its TrainingPortal or REST API, and retain only UDS-specific extensions that Educates cannot express cleanly.

## Decision

**Adopt Educates as the default workshop platform for new cluster-native scenarios only after proving the ingress and policy integration on a UDS Core cluster. Do not replace the current UDS Labs VM backend yet.**

The first technical spike should not start with workshop content. It should install Educates on a disposable UDS Core cluster and answer:

1. Can Educates' portal/session/application routes be emitted as UDS/Istio `VirtualService` resources, either through a fork or a translation controller?
2. Can UDS Core's tenant gateway handle the required per-session hostnames, TLS certificate, websocket behavior, long timeouts, and application routes?
3. Can the required route resources be scoped and cleaned up safely when a `WorkshopSession` expires?
4. Can Educates' required Kyverno workshop policies be replaced with equivalent Pepr policies, or can the workshop policy engine be disabled for the trusted-user deployment?
5. Can Educates session namespaces reach the UDS tenant gateway and UDS Core services under the intended RBAC and network policies?

A separate Contour installation remains the lowest-effort compatibility path, but it is not the only path. A UDS/Istio integration would avoid introducing a second ingress stack and is likely the better long-term direction for this project.

The strongest fit is not “Educates versus UDS Labs” as an all-or-nothing replacement. It is:

- Educates for flexible workshop definitions, variants, dashboard, portal, lifecycle, and verification;
- UDS Labs/KubeVirt for the specialized VM capability that makes the current UDS Core playground and nested environments reproducible.

The decisive proof point is whether the intended learning experience should teach users to package against a real UDS Core cluster rather than booting a private nested UDS Core VM. If the former, Educates is likely the better long-term foundation. If the latter remains central, use Educates only as a front door around a retained VM backend, or continue with the current specialized implementation.
