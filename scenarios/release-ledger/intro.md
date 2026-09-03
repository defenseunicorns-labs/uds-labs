# Package a Stateless Evidence App

Release Ledger records software releases and the evidence files associated with them. In this lab the application has no database or object store: release metadata lives in process memory and evidence content lives on the container filesystem.

You will author the UDS packaging layers around a pre-containerized application, deploy it to UDS Core, create a Release, and replace its pod. The Release will disappear because a pod is not durable storage.

**What's already running:** UDS Core (Keycloak, Istio, and Pepr) on a k3d cluster. The `release-ledger:dev` image is built during setup.

**What you'll write:** a Helm chart, UDS Package CR, `zarf.yaml`, bundle, and `tasks.yaml`.

> All files live in `/root/release-ledger`.
