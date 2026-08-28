// Package provider defines the VM backend abstraction the Lab Operator
// reconciles through (ADR-0011). KubeVirt on AKS is the planned implementation;
// the lifecycle interface keeps its orchestration independent from KubeVirt API
// details rather than promising a second cloud provider.
package provider

import (
	"context"

	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
)

// Result is the observed outcome of a Reconcile, mapped onto LabSession status.
type Result struct {
	// Phase is the provider's view of VM readiness. The operator may still hold
	// the session in Running until the HTTP readiness probe passes (two-phase
	// readiness), only then promoting to Ready.
	Phase labv1.LabSessionPhase
	// ServiceDNS is the in-cluster DNS name of the Service fronting the VM, set
	// once the VM is far enough along to have one.
	ServiceDNS string
	// Message is human-readable detail, surfaced on failures.
	Message string
}

// Provider creates and destroys the infrastructure backing one LabSession.
// Reconcile must be idempotent: it is called repeatedly until the session is
// Ready (and again on any change), and must converge existing objects rather
// than error on "already exists".
type Provider interface {
	// Reconcile ensures the VM and its supporting objects exist and reports
	// progress. It does not block waiting for readiness.
	Reconcile(ctx context.Context, ls *labv1.LabSession) (Result, error)
	// Teardown removes everything Reconcile created, including the session disk.
	// It must be idempotent and succeed if the objects are already gone.
	Teardown(ctx context.Context, ls *labv1.LabSession) error
	// TeardownCompute removes only the VMI, Service, and NetworkPolicy while
	// retaining the DataVolume/PVC. stopped is true only after the VMI no longer
	// exists, so the controller does not report Paused while compute is running.
	TeardownCompute(ctx context.Context, ls *labv1.LabSession) (stopped bool, err error)
}
