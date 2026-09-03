# Step 2 – Write the Helm chart

Helm is the Kubernetes packaging format that Zarf will deploy. Create `Chart.yaml`, `templates/deployment.yaml`, and `templates/service.yaml`. The `values.yaml` file with the image and domain is already provided.

```bash
cd /root/release-ledger
```

## Chart.yaml

```bash
cat > chart/Chart.yaml << 'EOF'
apiVersion: v2
name: release-ledger
description: Stateless Release Ledger application
type: application
version: 0.1.0
appVersion: dev
EOF
```

## Deployment

```bash
cat > chart/templates/deployment.yaml << 'EOF'
apiVersion: apps/v1
kind: Deployment
metadata:
  name: release-ledger
  namespace: {{ .Release.Namespace }}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: release-ledger
  template:
    metadata:
      labels:
        app: release-ledger
    spec:
      containers:
        - name: release-ledger
          image: {{ .Values.image }}
          ports:
            - containerPort: 8080
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
EOF
```

## Service

```bash
cat > chart/templates/service.yaml << 'EOF'
apiVersion: v1
kind: Service
metadata:
  name: release-ledger
  namespace: {{ .Release.Namespace }}
spec:
  selector:
    app: release-ledger
  ports:
    - port: 8080
      targetPort: 8080
EOF
```

The Service is internal to the cluster. The UDS Package CR in the next step exposes it through the tenant gateway.

## Render and check

```bash
uds zarf tools helm template test chart/
uds zarf tools helm template test chart/ 2>/dev/null | grep "kind:" | sort
```

You should see one Deployment and one Service with no rendering errors.
