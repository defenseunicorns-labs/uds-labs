# Upstream provenance

- Project: [Educates Training Platform](https://github.com/educates/educates-training-platform)
- Release: `3.8.0`
- License: Apache-2.0
- Renderer: `educates-linux-{amd64,arm64}` release binary, executed inside the validation/rendering container
- Configuration input: [`config/installer.yaml`](../config/installer.yaml)
- Generated output: [`chart/files/educates.yaml`](../chart/files/educates.yaml)

Regenerate deterministically with:

```bash
./scripts/render-manifests.sh
```

The script executes the official Educates CLI container but explicitly passes `--version 3.8.0`; the upstream `educates-cli:3.8.0` container reports a compiled default version of `develop`, so relying on its default would fetch the wrong installer bundle.

The only post-render transformation replaces the domain sentinel with `__EDUCATES_DOMAIN__`. The Helm wrapper replaces that sentinel from the package's `domain` value at deployment time.
