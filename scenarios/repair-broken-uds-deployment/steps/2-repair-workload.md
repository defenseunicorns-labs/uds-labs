# Step 2 – Repair the workload contract

The application image runs as a non-root user and starts with a filesystem-backed evidence store.
The Deployment currently sends it to a path the user cannot create, and it requests more capacity
than this lab needs.

Repair the Deployment and values without changing the image or application source:

1. Use `.Values.replicaCount`, `.Values.containerPort`, `.Values.evidenceDir`, and
   `.Values.resources` instead of hard-coded workload settings.
2. Add a pod `fsGroup` compatible with the image's UID and GID.
3. Mount an `emptyDir` at `.Values.evidenceDir` so temporary evidence is writable.
4. Use the named `http` container port for the probes.
5. Keep the evidence store explicitly temporary. This scenario does not add durable storage.

The values file already contains a small, valid lab baseline. The important repair is wiring those
values into the rendered Deployment.

Open the built-in Web IDE with the **IDE** button. In the file tree, open
`/root/release-ledger/chart/templates/deployment.yaml` and edit the file manually until it
matches this repaired workload contract. Do not run a replacement command in the terminal.

This is a display-only example of the complete file:

```yaml
# Copyright 2026 Defense Unicorns
# SPDX-License-Identifier: AGPL-3.0-or-later
apiVersion: apps/v1
kind: Deployment
metadata:
  name: release-ledger
  labels:
    app: release-ledger
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app: release-ledger
  template:
    metadata:
      labels:
        app: release-ledger
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
        - name: release-ledger
          image: {{ .Values.image | quote }}
          imagePullPolicy: IfNotPresent
          env:
            - name: EVIDENCE_DIR
              value: {{ .Values.evidenceDir | quote }}
          ports:
            - name: http
              containerPort: {{ .Values.containerPort }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
          volumeMounts:
            - name: evidence
              mountPath: {{ .Values.evidenceDir | quote }}
          readinessProbe:
            httpGet:
              path: /health
              port: http
          livenessProbe:
            httpGet:
              path: /health
              port: http
      volumes:
        - name: evidence
          emptyDir: {}
```

Save the file in the IDE. Then return to the Lab Terminal and render the chart:

```bash
uds zarf tools helm template release-ledger chart/ > /tmp/release-ledger-rendered.yaml
```

Inspect the rendered Deployment:

```bash
sed -n '/kind: Deployment/,/kind: Service/p' /tmp/release-ledger-rendered.yaml
```

## Verify

```bash
grep 'replicas: {{ .Values.replicaCount }}' chart/templates/deployment.yaml
```

```bash
grep 'fsGroup: 1000' chart/templates/deployment.yaml
```

```bash
grep 'value: {{ .Values.evidenceDir | quote }}' chart/templates/deployment.yaml
```

```bash
grep 'containerPort: {{ .Values.containerPort }}' chart/templates/deployment.yaml
```

```bash
grep 'toYaml .Values.resources' chart/templates/deployment.yaml
```

```bash
grep 'mountPath: {{ .Values.evidenceDir | quote }}' chart/templates/deployment.yaml
```

```bash
grep 'emptyDir: {}' chart/templates/deployment.yaml
```

```bash
grep 'port: http' chart/templates/deployment.yaml
```

The workload contract is repaired when each check succeeds.
