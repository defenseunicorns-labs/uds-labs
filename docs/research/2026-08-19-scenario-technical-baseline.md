# Scenario technical baseline

**Validated:** 2026-08-19

This note records the primary-source checks required before authoring the Postgres, MinIO, repair, and SSO scenarios in `docs/plans/2026-08-19-catalog-scenarios-context.md`.

## Pinned lab toolchain

- The VM base image pins UDS CLI `v0.35.1` in `packer/scripts/base.sh`.
- UDS CLI `v0.35.1` depends on Zarf `v0.83.0`. Source: [`uds-cli` v0.35.1 `go.mod`](https://github.com/defenseunicorns/uds-cli/blob/v0.35.1/go.mod).
- Scenario syntax should therefore be validated against UDS CLI `v0.35.1` and Zarf `v0.83.0`, not the older CLI currently installed on the development host.

## Configuration syntax

- UDS bundle `overrides.<component>.<chart>.values` are fixed at bundle creation. `variables` remain deploy-time configurable and can be supplied through `uds-config.yaml`, `UDS_` environment variables, or `--set`. Source: [UDS CLI v0.35.1, Use Bundle Overrides](https://github.com/defenseunicorns/uds-cli/blob/v0.35.1/docs/how-to-guides/use-bundle-overrides.mdx).
- Zarf Package Values are **beta** in Zarf `v0.83.0`, not alpha. They provide structured data, schemas, Go templating, and chart mappings through `sourcePath` and `targetPath`. Source: [Zarf v0.83.0, Package Values](https://github.com/zarf-dev/zarf/blob/v0.83.0/site/src/content/docs/ref/package-values.mdx).
- Current Zarf guidance recommends migrating legacy Variables to Package Values. Variables still work and remain relevant to UDS bundle deploy-time override workflows. Source: [Zarf v0.83.0, Variables to Values](https://github.com/zarf-dev/zarf/blob/v0.83.0/site/src/content/docs/best-practices/variables-to-values.mdx).
- For these scenarios, use the published package's exposed bundle override contract. Do not rewrite upstream packages merely to demonstrate Package Values.

## UDS Package networking

- `network.expose[].port` is the Service port. `targetPort` is separately supported and is required when the Service port differs from the selected pod port so generated policy is correct. Source: [UDS Core Package CR reference](https://github.com/defenseunicorns/uds-core/blob/main/docs/reference/operator-and-crds/packages-v1alpha1-cr.md).
- Every `network.allow` entry needs a direction, workload selector, and a remote target such as `remoteNamespace`/`remoteSelector` or `remoteGenerated`. The UDS Operator generates Kubernetes NetworkPolicies and Istio resources from these declarations. Source: [Define network access](https://github.com/defenseunicorns/uds-core/blob/main/docs/how-to-guides/networking/define-network-access.mdx).
- Cross-namespace application-to-database traffic needs reciprocal declarations: application egress and database ingress. The Postgres package exposes database-side ingress through `postgresql.ingress`.

## Postgres package baseline

Pin the public package to:

```yaml
repository: ghcr.io/uds-packages/postgres-operator
ref: 1.15.1-uds.6-upstream
```

Primary sources:

- [Release](https://github.com/uds-packages/postgres-operator/releases/tag/1.15.1-uds.6-upstream)
- [Configuration](https://github.com/uds-packages/postgres-operator/blob/1.15.1-uds.6-upstream/docs/configuration.md)
- [Bundle example](https://github.com/uds-packages/postgres-operator/blob/1.15.1-uds.6-upstream/bundle/uds-bundle.yaml)

Confirmed contract:

- Configure the built-in `Postgresql` resource at `postgres-operator.uds-postgres-config` path `postgresql`.
- `postgresql.users` uses `{namespace}.{username}` keys. With cross-namespace Secret creation enabled by the package, the operator creates the user's credential Secret in that namespace.
- The generated Secret name follows the Zalando operator username/cluster convention. The scenario must confirm the exact name and keys in a live deployment before fixture finalization rather than encode an assumption from an older package.
- `postgresql.ingress` declares database-side ingress and fixes the database selector to `cluster-name: pg-cluster`.
- Lab sizing can set one instance, a small volume, and explicit requests/limits. The rendered production comparison should use larger storage and multiple instances without deploying them.

Application credentials should use direct `secretKeyRef` bindings when the application accepts discrete values. The Reference Package's `chart/templates/postgres-secret.yaml` uses Helm `lookup` to turn generated fields into one connection string. That can race asynchronous operator reconciliation and should be taught as a compatibility adapter, not the default pattern.

## MinIO package baseline

Pin the public package to:

```yaml
repository: ghcr.io/uds-packages/minio-operator
ref: 7.1.1-uds.25-upstream
```

Primary sources:

- [Release](https://github.com/uds-packages/minio-operator/releases/tag/7.1.1-uds.25-upstream)
- [Configuration](https://github.com/uds-packages/minio-operator/blob/7.1.1-uds.25-upstream/docs/configuration.md)
- [Bundle example](https://github.com/uds-packages/minio-operator/blob/7.1.1-uds.25-upstream/bundle/uds-bundle.yaml)

Confirmed contract:

- Configure application buckets and scoped users at `minio-operator.uds-minio-config` path `apps`.
- Each app entry supplies `name`, application `namespace`, workload `remoteSelector`, `bucketNames`, optional policy, and `copyPassword` configuration.
- `copyPassword` can create an application-namespace Secret with scenario-selected Secret name and key names. This supports direct `secretKeyRef` bindings without committing credentials.
- Built-in tenant storage is overridden at `minio-operator.minio-tenant` path `tenant.pools`.
- `additionalNetworkAllow` is scoped to the built-in tenant Package. `operatorAdditionalNetworkAllow` is separately scoped to the operator Package.

The exact copied Secret content and MinIO endpoint must still be observed in a live deployment before the scenario verifier is finalized.

## Authservice baseline

- Authservice is enabled by adding `sso[].enableAuthserviceSelector` to the application Package CR. Source: [Protect applications with Authservice](https://github.com/defenseunicorns/uds-core/blob/main/docs/how-to-guides/identity-and-authorization/protect-apps-with-authservice.mdx).
- The selector must match both pod labels and the Kubernetes Service selector. A mismatch commonly produces 503 responses.
- Authservice redirect URIs cannot use a root path; use a specific path such as `/login`.
- The UDS Operator creates the Keycloak client, Authservice configuration, `RequestAuthentication`, and `AuthorizationPolicy` resources.
- For native application OIDC, the generated client Secret defaults to `sso-client-<clientId>` and can be reshaped with `secretConfig`. Source: [Register and customize SSO clients](https://github.com/defenseunicorns/uds-core/blob/main/docs/how-to-guides/identity-and-authorization/register-and-customize-sso-clients.mdx).

## Still requires runtime validation

The source-level contracts are sufficient for catalog work and fixture design. Before publishing dependency-heavy scenarios, run them in a `large` Lab Session to record:

1. Exact Postgres generated Secret name and key set.
2. Exact MinIO copied Secret values and in-cluster endpoint.
3. Package pull time, reconciliation time, disk growth, and retry behavior.
4. Generated network resource names/selectors used by verification.
5. Authservice redirect and authenticated-access behavior against the UDS Core version baked into the VM image.
