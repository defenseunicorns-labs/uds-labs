# CDI package verification

Verified 2026-08-29 against the primary source repository, [`uds-packages/containerized-data-importer`](https://github.com/uds-packages/containerized-data-importer), at commit [`43131898febb09f2a2bad02df84b1e23017c0101`](https://github.com/uds-packages/containerized-data-importer/commit/43131898febb09f2a2bad02df84b1e23017c0101). This is `main` and is tagged `1.65.0-uds.5-upstream`.

## Result

`bundle/uds-bundle.yaml` already pins that newest published tag:

```yaml
repository: ghcr.io/uds-packages/cdi
ref: 1.65.0-uds.5-upstream
```

The latest CDI change adds `workloadNamespaces`, which emits scoped Istio traffic-override exemptions for CDI host-assisted clone and upload pods outside the CDI namespace. Labs correctly supplies:

```yaml
path: workloadNamespaces
name: WORKLOAD_NAMESPACES
default: [uds-labs-vms]
```

The other placement overrides are also valid current chart values:

- `operator.nodePlacement`
- `cdi.infra`
- `cdi.workloads`

Two obsolete overrides were removed: `uploadProxy.expose` and `network.importerEgress.enabled`. Neither exists in the current package's `chart/values.yaml`; Helm would silently ignore them.

## Sources

- CDI tag/main resolution: `git ls-remote --tags` and `git ls-remote refs/heads/main` against the repository above.
- Current values: [`chart/values.yaml`](https://github.com/uds-packages/containerized-data-importer/blob/43131898febb09f2a2bad02df84b1e23017c0101/chart/values.yaml)
- Workload exemption rendering: [`chart/templates/uds-exemption.yaml`](https://github.com/uds-packages/containerized-data-importer/blob/43131898febb09f2a2bad02df84b1e23017c0101/chart/templates/uds-exemption.yaml)
- CDI CR placement rendering: [`chart/templates/cdi.yaml`](https://github.com/uds-packages/containerized-data-importer/blob/43131898febb09f2a2bad02df84b1e23017c0101/chart/templates/cdi.yaml)
