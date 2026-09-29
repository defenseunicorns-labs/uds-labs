# Repair a Broken UDS Deployment

A customer has handed over a package that was reported as deployable. Your job is to review the
package, find the failures, and make the smallest declarative fixes needed to deploy it through UDS
Core.

The package contains a pre-containerized Release Ledger application and a complete Zarf package,
UDS Package configuration, bundle, and development task. The application source is not part of the
repair scope.

You will work through the failure in layers:

1. Collect evidence before editing.
2. Repair the workload's configuration contract.
3. Trace and repair Service and UDS exposure routing.
4. Correct the network policy selector.
5. Rebuild the Zarf package and UDS bundle, then verify real application behavior through ingress.

The package and bundle are not the same thing. The package is the atomic deployable artifact; the
bundle composes packages and deployment configuration. The Package CR is a Kubernetes resource
inside the UDS package that declares the application's UDS Core integrations.

> [!NOTE]
> This lab uses temporary filesystem storage. The repair makes that behavior explicit; it does not
> turn temporary evidence into durable production storage.

All work lives in `/root/release-ledger`.
