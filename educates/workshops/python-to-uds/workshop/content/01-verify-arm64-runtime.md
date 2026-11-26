# Verify the arm64 runtime

Educates exposes the node architecture as `PLATFORM_ARCH`. The Linux kernel reports the same architecture as `aarch64`.

```terminal:execute
command: |
  printf 'Educates: %s\nKernel:   %s\n' "$PLATFORM_ARCH" "$(uname -m)"
```

The workshop image includes the same UDS CLI version used by the current VM image.

```terminal:execute
command: uds version
```

Educates supplies a private Docker daemon for this session. It must also be native arm64 rather than an amd64 daemon running through emulation.

```terminal:execute
command: |
  docker version --format 'Docker server: {{.Server.Arch}} {{.Server.Version}}'
```

The session receives Kubernetes credentials scoped by Educates. Display the namespace and verify basic access.

```terminal:execute
command: |
  kubectl config view --minify -o jsonpath='{..namespace}'; echo
  kubectl auth can-i get pods
```

Run the complete checkpoint:

```examiner:execute-test
name: arm64-runtime
title: Verify the native arm64 UDS workshop runtime
timeout: 30
```
