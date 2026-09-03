# Step 1 – The Release Ledger application

Your working directory is `/root/release-ledger`. The application source, dependencies, and Dockerfile are already here so you can focus on the UDS packaging layers.

```bash
cd /root/release-ledger
sed -n '1,240p' app.py
cat requirements.txt
cat Dockerfile
```

Release Ledger provides:

- `GET /health` — readiness response
- `POST /api/releases` — create a Release
- `GET /api/releases` — list Releases
- `POST /api/releases/<id>/evidence` — upload Evidence
- `GET /api/releases/<id>/evidence` — list Evidence
- `GET /api/evidence/<id>/content` — download Evidence
- `DELETE /api/evidence/<id>` — delete Evidence

The application is deliberately prebuilt. You will not change Python code in this lab.

The default configuration is:

```text
STORAGE_MODE=memory
EVIDENCE_STORAGE=filesystem
```

Release metadata is held in process memory and evidence content is written to the container filesystem. Both are intentionally ephemeral and disappear when Kubernetes replaces the pod.

The image is built during setup. Check when it is available:

```bash
docker image ls release-ledger:dev
```

## Verify

```bash
ls app.py requirements.txt Dockerfile
