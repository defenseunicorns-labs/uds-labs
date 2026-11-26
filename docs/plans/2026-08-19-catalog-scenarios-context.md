# UDS Labs Catalog and Scenario Refinement — Saved Context

**Saved:** 2026-08-19
**Branch at save:** `chore/hetzner-local-dev-cleanup`
**Commit at save:** `cc56ae7`
**Status:** Product and scenario contracts are settled. The first catalog increment is implemented; dependency-heavy scenario authoring and runtime validation remain.

## Objective

Refine the UDS Labs catalog and add a small number of stronger learning scenarios for ISVs, customers, and engineers at different readiness levels. Use official UDS and Zarf documentation as canonical technical references while incorporating sanitized lessons from real customer onboarding and deployment work.

This effort is intentionally narrow. The current pass covers:

1. The catalog experience.
2. A few high-value scenarios.
3. A possible optional readiness assessment.

Do not expand this pass into an LMS, durable learner profiles, certification, or a broad curriculum platform. The longer-term direction may become a full LMS, but that is explicitly deferred.

## Agreed Product Decisions

### Promise and audience

- Primary promise: move a learner to their **next independently demonstrable capability**.
- Understanding concepts and learning to troubleshoot support that promise; they are not separate product goals.
- Optimize the first experience for an **ISV application developer who also owns DevOps**.
- Internal DU engineers can use the same material as train-the-trainer content.

### Scope boundaries

- Provide a shared, product-neutral Zarf/UDS foundation.
- Defer the UDS Army-specific CI, CAT, sandbox, and compliance track from the current implementation. Preserve it as future work in this plan; do not add its catalog cards or readiness questions yet.
- Start hands-on UDS material once the learner has a containerized application and basic Kubernetes/Helm literacy.
- Route learners below that prerequisite floor to curated external resources rather than recreating Docker or Kubernetes training.
- Design content to be public-safe from the beginning. Sanitize customer-derived patterns; keep internal infrastructure and customer-specific details out of public scenarios.

### Near-term product shape

- UDS Labs should become a lightweight **learning front door**, not remain only a VM runtime.
- Interactive labs remain the key modality, but the catalog may also present curated resources.
- A full LMS is an eventual goal, not part of this pass.

## Agreed Catalog Direction

Keep the near-term catalog intentionally simple:

1. **Scenarios** — guided, hands-on labs with a defined outcome.
2. **Playgrounds** — open exploration.

Show compact cards with title, short description, difficulty, and expected duration. Keep richer outcome and prerequisite metadata in the scenario contract without presenting it on every card. Defer readiness routing and external-resource cards until the scenario library is large enough to need them.

A future **UDS Army Journey** may curate CI, CAT, sandbox, and compliance resources. Keep that track in planning documents only for now. Avoid building paths, transcripts, credentials, durable progress, or other LMS machinery during this pass.

## Deferred Readiness Assessment

The assessment was implemented in the first catalog pass, then removed after reviewing the interface. Preserve the agreed shape for a later catalog with enough content to require routing:

- Optional and approximately two minutes.
- Client-side only.
- Initially covers containers, Kubernetes/Helm, and UDS/Zarf.
- CI, CAT, compliance, and UDS Army questions are deferred with the UDS Army track.
- Recommends where to begin in the catalog.
- Does not produce a single readiness score.
- Does not require an account or retain a transcript.

The Warhacker readiness assessment is useful source material, but its content should be updated against current canonical docs and UDM workflows.

## Existing Catalog Baseline

The current catalog is a flat grid containing three guided scenarios and two playgrounds:

### Guided scenarios

- `uds-package-quickstart`
- `uds-reference-package`
- `python-to-uds`

### Playgrounds

- `playground-tools`
- `playground-uds-core`

The existing guided scenarios are concentrated on package authoring and understanding package structure. They overlap somewhat and provide less depth on real integration and troubleshooting failures.

## Agreed Scenario Priorities

The next high-value scenario themes are:

1. **Repair a Broken UDS Deployment**
   - Diagnose UDS `Package` status.
   - Correct service/target-port mismatches.
   - Repair ingress and egress rules.
   - Inspect generated policies and resources.

2. **Secure an Existing App with UDS SSO**
   - Add Keycloak/Authservice integration.
   - Understand generated client configuration and Secrets.
   - Troubleshoot redirects and client configuration.

3. **Make an App Production-Shaped**
   - Highest priority.
   - Focus on the recurring difficulty of integrating Postgres Operator, MinIO Operator, configuration, Secrets, resources, and bundle composition.

CI, CAT, sandbox, compliance, and other UDS Army resources are deferred from the current catalog. When that track is revisited, begin with curated resources rather than executable labs because realistic exercises depend on credentials and external systems.

## Production-Shaped Experience

Rather than create one oversized lab, split this into two focused scenarios that share the same fictional sample application.

### Scenario A: Connect an App to Postgres

Teach and verify:

- The application package versus deployment bundle boundary.
- Postgres Operator configuration in the bundle.
- A separate backing-service namespace.
- Operator-generated application credentials.
- Direct application consumption through `secretKeyRef`.
- Non-secret runtime configuration through a ConfigMap.
- Network rules on both the application and database sides.
- An actual database write/read as the end-state verification.

### Scenario B: Add Object Storage and Production Constraints

Teach and verify:

- MinIO tenant configuration.
- Scoped application user and bucket creation.
- Copying credentials into the application namespace.
- Direct application consumption through `secretKeyRef`.
- MinIO endpoint and network configuration.
- CPU/memory requests and limits.
- Storage sizing and environment-specific overrides.
- An actual object put/get/delete as the end-state verification.

### Shared application contract

Both scenarios use a fictional **Release Ledger** application:

- A Release has a UUID, application name, version, environment, and creation time.
- Evidence metadata includes release ID, filename, object key, size, SHA-256 checksum, and creation time.
- A Release may have multiple evidence files.
- The UI supports creating and listing Releases and uploading, downloading, and deleting Evidence.
- Postgres stores Releases and Evidence metadata; filesystem or MinIO stores Evidence content.
- The application exposes a JSON API so verification is deterministic.
- Application source is inspectable but not edited by the learner.

Scenario A starts with in-memory Release storage and demonstrates data loss after pod replacement. It ends with operator-managed Postgres and durable Release records.

Scenario B starts from the completed Postgres baseline with filesystem-backed Evidence and demonstrates file loss after pod replacement. It ends with MinIO-backed Evidence that survives replacement.

Use a resource-constrained lab configuration at runtime. Render and compare a production example to show different storage, replica, and resource values without deploying an oversized production profile.

## Exercise and Verification Contract

- Give learners an existing production-shaped repository with focused configuration gaps rather than asking them to author boilerplate from scratch.
- Use explicit TODOs in the Postgres and MinIO scenarios. Reserve hidden defects for **Repair a Broken UDS Deployment**.
- Keep application source, base Helm structure, images, and operator packages complete.
- Learners edit bundle dependencies and overrides, backing-service resources, ConfigMaps, Deployment `secretKeyRef` bindings, and UDS network rules on both sides.
- Advertise each production-shaped scenario as 45 minutes and target about 30 minutes of active work.
- Pull public Postgres and MinIO packages from their OCI repositories through `uds-bundle.yaml` at learner runtime. Pin exact versions; do not pre-stage the packages. Add connectivity and disk preflight checks, progress messaging, and retry guidance.
- Run dependency-heavy scenarios with the `large` resource tier.

Verification has three layers:

1. **Static** — no credential literals, correct ConfigMap and Secret references, and expected resource configuration.
2. **Platform** — backing service and UDS Package resources are ready, with intended generated network resources.
3. **Behavioral** — data survives application pod replacement; object content matches by checksum; deleting Evidence removes the object.

Do not initially add an active negative-connectivity probe from an unauthorized workload. Verify both network declarations, generated policy resources, and the successful authorized transaction.

## Repair and SSO Scenarios

**Repair a Broken UDS Deployment** reuses Release Ledger with layered deployment defects: a non-root filesystem permission failure, a Service/container target-port mismatch, an incorrect UDS exposure reference, and an incorrect or missing database network declaration on one side. The fixture has no revealing TODO comments. Guided steps offer progressively stronger hints and focused checks; a separate aggregate verifier proves the complete repaired state. This keeps future interview use possible without adding assessment mode, scoring, tracking, or anti-cheat features.

**Secure an Existing App with UDS SSO** uses Release Ledger in stateless mode so Postgres does not distract from the lesson. It teaches UDS Authservice rather than native application OIDC: configure the UDS Package SSO declaration, inspect the generated client and Secret, troubleshoot redirects/client configuration, and prove unauthenticated redirect plus authenticated access.

## Configuration and Secret Mental Model

The labs should explicitly trace one non-secret and one secret through the full configuration chain:

```text
uds-config
  → bundle variable or override
  → package value
  → Helm value
  → ConfigMap or Secret
  → pod environment/mount
```

Teach these boundaries:

- ConfigMaps contain non-sensitive runtime configuration.
- Secrets contain sensitive runtime values.
- Git contains Secret names and key names, not production secret values.
- Operators generate or copy application-scoped credentials.
- Applications consume credentials directly with `secretKeyRef` where possible.
- Bundle overrides compose and bind the application to backing services.
- The same application package should be configurable for different deployment environments without modification.

Current canonical UDS guidance prefers Zarf Package Values for package configuration and retains Zarf Variables mainly for compatibility. Validation against the lab's pinned UDS CLI `v0.35.1` / Zarf `v0.83.0` found Package Values are beta. Published dependency packages still expose bundle `values` and `variables` contracts that scenarios should consume rather than rewrite.

## Real-World Patterns to Borrow

Use `~/src/github.com/defenseunicorns-udm/uds-dot-mil/customerDeployments` as evidence of recurring patterns, not as content to copy directly.

### Exia Labs and PetraData patterns

- Postgres and MinIO CR packages live in separate owned namespaces.
- Postgres credentials are generated into the application namespace.
- MinIO creates scoped users/buckets and copies credentials into application-specific Secrets.
- Application workloads consume credentials through `secretKeyRef` or configurable existing-Secret contracts.
- Bundle overrides bind hosts, namespaces, selectors, Secret names/keys, storage, and resources.
- Network policy must be configured on both sides of a backing-service connection.

Relevant references:

- `defenseunicorns-udm/uds-dot-mil/customerDeployments/exia-labs/customer-deployment.yaml`
- `defenseunicorns-udm/uds-dot-mil/customerDeployments/petradata/customer-deployment.yaml`

### Fulcrum pattern

- Disable an application chart's bundled database.
- Substitute an operator-managed Postgres instance at the bundle layer.
- Make host and credential-Secret bindings configurable.

Relevant reference:

- `defenseunicorns-udm/uds-dot-mil/customerDeployments/fulcrum-platform/customer-deployment.yaml`

### Other lessons

- Customer deployment history shows repeated fixes involving resource limits, operator versions, Secret references, database connectivity, namespace ownership, and backing-service substitutions.
- `customerDeployments/` contains effective platform deployment state and historical workarounds. It is not a canonical public template.
- Remove customer names, internal registry details, platform mutation details, and real Secret names from training fixtures.

## Reference Package Guidance

Use `~/src/github.com/uds-packages/reference-package` for its package structure and conceptual boundaries:

- Dependencies belong in the bundle.
- The application package contains the application and UDS integration.
- The UDS `Package` CR declares network, SSO, and monitoring needs.
- The app consumes a database connection from a Kubernetes Secret.

However, do not present its current render-time Postgres adapter as the preferred production pattern. `chart/templates/postgres-secret.yaml` uses Helm `lookup` to read an operator-created Secret and construct a DSN during rendering. That can race asynchronous operator reconciliation.

The agreed scenario should explicitly improve on that pattern by using:

- Direct Secret references where the application supports discrete credentials; or
- A bounded runtime adapter/init process when the application only accepts a DSN.

It is acceptable for the lab to explain this limitation rather than blindly treating every reference-package implementation detail as canonical.

## Canonical Sources

When implementation details disagree, use these as the technical source of truth:

- <https://docs.defenseunicorns.com>
- <https://docs.zarf.dev>

Useful local/source references inspected during discovery:

- `defenseunicorns-udm/warhacker-starter-kit/docs/00-readiness-self-assessment.md`
- `defenseunicorns-udm/warhacker-starter-kit/docs/01-personas-and-paths.md`
- `defenseunicorns-udm/warhacker-starter-kit/docs/gotchas/README.md`
- `defenseunicorns-udm/udm-common/CONTEXT.md`
- `defenseunicorns-udm/udm-common/README.md`
- `defenseunicorns-udm/sandbox-module/README.md`
- `defenseunicorns-udm/sandbox-module/compatibility/README.md`
- `defenseunicorns-udm/uds-dot-mil/docs/plans/isv-deployment-compatibility.md`
- `defenseunicorns-udm/uds-dot-mil/customerDeployments/`
- `uds-packages/reference-package/`

Warhacker is useful for persona-aware routing and real-world gotchas, but it is not authoritative and already contains stale CAT/CI details.

## Catalog Implementation Decisions

- Retire `uds-package-quickstart` because its setup and nginx packaging overlap stronger material.
- Retain and rename `python-to-uds` to **Package a Stateless App for UDS**.
- Retain and revise `uds-reference-package` as **Explore UDS Package Structure**, removing claims that every implementation detail is canonical and explaining its Postgres render-time adapter limitation.
- Use a central `catalog.yaml` to define sections, ordering, scenario references, and external resources.
- Add only intrinsic `outcome` and `prerequisites` fields to scenario metadata.
- Prerequisites are advisory and never block launch.
- Defer the client-side readiness assessment. If restored, it gives one primary and up to two secondary recommendations with explanations and does not calculate a score.
- Do not include the UDS Army track in the current catalog or assessment.

Delivery should remain incremental:

1. Catalog sections, metadata, readiness assessment, public prerequisite resources, and existing-scenario cleanup.
2. Postgres scenario and shared Release Ledger fixture.
3. MinIO scenario extending the fixture.
4. Repair scenario.
5. SSO scenario.

## Remaining Technical Research Before Authoring

Source-level research is captured in `docs/research/2026-08-19-scenario-technical-baseline.md`.

Completed:

1. Revalidated syntax against pinned UDS CLI `v0.35.1`, Zarf `v0.83.0`, and current first-party docs.
2. Selected public OCI baselines: Postgres `1.15.1-uds.6-upstream` and MinIO `7.1.1-uds.25-upstream`.
3. Validated UDS Package network and Authservice syntax from UDS Core source docs.
4. Selected public prerequisite and canonical-documentation links for the catalog.

Still required before publishing dependency-heavy scenarios:

1. Observe exact operator-generated and copied Secret contents in a live Lab Session.
2. Measure public package pull and deployment timing inside a `large` Lab Session.
3. Verify generated network resources and Authservice behavior against the UDS Core version baked into the VM image.
4. Define the shared fixture layout and exact learner-edited files without duplicating its source across scenarios.

## Deferred UDS Army Track

Keep these notes for a later pass; do not surface them in the current product:

- A clearly labeled UDS Army Journey rather than mixing Army-specific workflow into product-neutral UDS material.
- Curated CI, CAT, sandbox, and compliance resources before attempting executable labs.
- Readiness questions for CI/CAT and compliance needs.
- Honest access labeling for any restricted resources.

Do not reopen LMS architecture or the deferred UDS Army track unless the user explicitly expands scope.

## Implementation Progress

Completed in the first catalog increment:

- Added central `scenarios/catalog.yaml`, then simplified it to **Scenarios** and **Playgrounds** after interface review.
- Added intrinsic `outcome` and advisory `prerequisites` scenario metadata while keeping cards compact.
- Added `/api/catalog` while retaining the flat scenario API for lab and admin consumers.
- Removed the initial readiness assessment and external-resource cards; both are deferred.
- Retired `uds-package-quickstart`.
- Renamed `python-to-uds` to **Package a Stateless App for UDS**.
- Renamed and revised `uds-reference-package` to **Explore UDS Package Structure**, including the render-time Postgres adapter limitation.
