# Scenario Preview

Scenario Preview is a small containerized authoring environment for walking through a scenario without Kubernetes, KubeVirt, nested virtualization, or a local UDS Core installation. It runs on Docker Desktop or Podman hosts using the image architecture native to the host, including Apple Silicon.

It is a scenario smoke test, not a replacement for live UDS E2E validation. The preview runs ordinary filesystem commands against a disposable fixture and simulates only the UDS, Docker, kubectl, and watch commands that a scenario explicitly teaches. It never creates a cluster or talks to a Docker socket.

## Run the first preview

From the repository root, run:

```bash
uds run preview:serve
```

Open <http://localhost:8080>. The preview starts the **Package a Stateless App for UDS** fixture in `/root/myapp`, displays the learner-facing steps, permits editing fixture files, runs documented commands, and verifies each step's file-level outcome.

Pass another local port or scenario identifier when more preview-enabled scenarios are available:

```bash
uds run preview:serve --with port=8081 --with scenario=python-to-uds
```

The container is removed when the command stops. Re-running it always starts from a clean fixture.

## Authoring contract

A preview-enabled `scenario.yaml` declares a fixture, its container workspace, and the file-level assertions needed for each step:

```yaml
preview:
  fixture: preview/fixture
  workspace: /root/myapp
  checks:
    - step: 2
      path: chart/templates/deployment.yaml
      contains: ["kind: Deployment", "containerPort: 8080"]
```

`fixture` is copied into the writable workspace at preview startup. Checks deliberately inspect learner artifacts instead of reproducing UDS or Kubernetes behavior. Add a simulation only for a documented command with a clear learning outcome; do not turn Scenario Preview into a general-purpose cluster emulator.

Validate all scenario metadata and preview contracts without building an image:

```bash
go run ./cmd/scenario-preview validate --scenarios ./scenarios
```
