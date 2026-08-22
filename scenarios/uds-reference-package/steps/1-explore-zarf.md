# Step 1 – Explore the reference package

UDS Core is already running on this cluster. Your job in this lab is to understand how a real UDS package is structured, then deploy it.

The [UDS Reference Package](https://github.com/uds-packages/reference-package) is a maintained example from Defense Unicorns. Use it to understand package boundaries and current integration patterns, then validate the details against the versions and needs of your own application.

The repo has been pre-cloned for you. Move into it:

```bash
cd /root/reference-package
```

In your own environment you'd clone it with:
```bash
git clone --depth 1 https://github.com/uds-packages/reference-package
```

## What's in the repo

```bash
ls -1
```

Key directories:

| Path | Purpose |
|------|---------|
| `zarf.yaml` | The Zarf package definition — what gets built and shipped |
| `bundle/` | UDS bundle definitions for deployment |
| `chart/` | Helm chart for the application |
| `values/` | Flavor-specific Helm value overrides |
| `tasks.yaml` | UDS task runner definitions (`uds run dev`, etc.) |

## Read zarf.yaml

```bash
cat zarf.yaml
```

Notice what is **not** in `zarf.yaml`:

- No Postgres operator
- No cert-manager
- No cluster setup
- No Keycloak deployment

`zarf.yaml` contains the reference package application and its UDS configuration chart. Shared infrastructure dependencies stay out of the application package so a deployment bundle can select, order, and configure them for the target environment.

## Check the image reference

```bash
grep -A3 "images:" zarf.yaml
```

The container image is pulled from `ghcr.io` at build time and bundled into the Zarf archive. When deployed in an air-gapped environment, no external registry is needed — everything is in the `.tar.zst` package file.

## Verify

```bash
ls chart/ values/ bundle/
```
