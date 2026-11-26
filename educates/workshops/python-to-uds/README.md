# Package a Stateless App for UDS — Educates spike

This is a deliberately narrow first slice of replacing the UDS Labs KubeVirt/CDI runtime with Educates. It proves the native arm64 workshop container, UDS CLI, private Docker daemon, Kubernetes credentials, and arm64 application build before attempting shared UDS Core deployment.

## Requirements

- Educates 3.8.0 installed on a disposable native-arm64 Kubernetes cluster.
- An OCI registry reachable by the operator and cluster nodes.
- Docker Buildx with access to that registry.
- The Educates 3.8.0 CLI for publishing and deploying the workshop.

Educates Docker support uses a privileged sidecar. Do not expose this workshop to untrusted users on a persistent shared cluster.

## Test on an Apple Silicon Mac

Allocate at least 12 GiB of memory to Docker Desktop before creating the local cluster. The Educates platform, editor, workshop container, and Docker sidecar otherwise may not schedule together.

Install the pinned Educates CLI:

```bash
curl -fsSL -o educates \
  https://github.com/educates/educates-training-platform/releases/download/3.8.0/educates-darwin-arm64
chmod +x educates
sudo mv educates /usr/local/bin/educates
educates version
```

If macOS blocks the unsigned binary, allow it under **System Settings → Privacy & Security**, or remove the quarantine attribute with `xattr -d com.apple.quarantine /usr/local/bin/educates`.

Validate the workshop without creating a cluster:

```bash
./scripts/validate.sh
```

Create Educates' disposable local Kind cluster and confirm that it is arm64:

```bash
educates local cluster create
kubectl get nodes \
  -o custom-columns=NAME:.metadata.name,ARCH:.status.nodeInfo.architecture
```

The node must report `arm64`. The local cluster provides a registry at `localhost:5001`. Build and publish the spike into it:

```bash
export IMAGE_REPOSITORY=localhost:5001
export WORKSHOP_VERSION=arm64-spike
./scripts/build-image.sh
./scripts/publish-workshop.sh
educates deploy-workshop -f workshop-arm64-spike.yaml
educates browse-workshops
```

When finished, remove the disposable cluster:

```bash
educates local cluster delete
```

## Validate locally

Validation renders the workshop with the checksum-pinned Educates 3.8.0 CLI. It does not require an Educates cluster and supports Linux or macOS on amd64 or arm64.

```bash
./scripts/validate.sh
```

## Build the arm64 workshop image

Authenticate Docker to the target registry, then run:

```bash
export IMAGE_REPOSITORY=ghcr.io/your-org
export WORKSHOP_VERSION=arm64-spike
./scripts/build-image.sh
```

`TARGET_PLATFORMS` defaults to `linux/arm64`. Set it to `linux/amd64,linux/arm64` when a multi-architecture image is desired.

## Publish and deploy

```bash
export IMAGE_REPOSITORY=ghcr.io/your-org
export WORKSHOP_VERSION=arm64-spike
./scripts/publish-workshop.sh
educates deploy-workshop -f workshop-arm64-spike.yaml
educates browse-workshops
```

## Current boundary

This slice stops after building and checking `myapp:dev`. The next slice will answer the harder shared-cluster question: which permissions and namespace/hostname substitutions are required for Zarf package creation and deployment against a host-cluster UDS Core installation.
