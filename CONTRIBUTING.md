# Contributing to UDS Labs

Thank you for contributing to UDS Labs.

## Development

Run the standard package checks before opening a pull request:

```bash
go test ./...
uds run dry-run
uds run lint:all
```

Use `uds run dev` for the k3d package-development loop. Use
`uds run local-e2e` only on a KVM-capable host when validating the full KubeVirt
session lifecycle.

## Pull requests

- Keep changes focused and explain the user-visible or operational effect.
- Add or update tests at the relevant public seam.
- Update documentation when task names, bundle behavior, or deployment inputs change.
- Use a conventional commit-style PR title.
- Do not commit local package archives, deployment configuration, credentials, or VM image outputs.

UDS package publication and UDS Proving Ground onboarding are performed by the
repository workflows after review and merge.
