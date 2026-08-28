// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package controller

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
	"github.com/defenseunicorns-labs/uds-labs/internal/provider"
)

type testProvider struct {
	reconcile       func(context.Context, *labv1.LabSession) (provider.Result, error)
	teardown        func(context.Context, *labv1.LabSession) error
	teardownCompute func(context.Context, *labv1.LabSession) (bool, error)
}

func (p testProvider) Reconcile(ctx context.Context, ls *labv1.LabSession) (provider.Result, error) {
	if p.reconcile == nil {
		return provider.Result{}, nil
	}
	return p.reconcile(ctx, ls)
}

func (p testProvider) Teardown(ctx context.Context, ls *labv1.LabSession) error {
	if p.teardown == nil {
		return nil
	}
	return p.teardown(ctx, ls)
}
func (p testProvider) TeardownCompute(ctx context.Context, ls *labv1.LabSession) (bool, error) {
	if p.teardownCompute == nil {
		return true, nil
	}
	return p.teardownCompute(ctx, ls)
}

func controllerTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := labv1.AddToScheme(s); err != nil {
		t.Fatalf("add LabSession scheme: %v", err)
	}
	return s
}

func TestReconcileFinalizerWaitsForWatchEvent(t *testing.T) {
	ctx := context.Background()
	s := controllerTestScheme(t)
	ls := &labv1.LabSession{ObjectMeta: metav1.ObjectMeta{Name: "session", Namespace: "uds-labs-vms"}}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(ls).Build()
	r := &LabSessionReconciler{Client: c, Scheme: s, Provider: testProvider{}}

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
		Name: ls.Name, Namespace: ls.Namespace,
	}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if result.Requeue || result.RequeueAfter != 0 {
		t.Fatalf("finalizer update must wait for its watch event, got requeue result %+v", result)
	}

	got := &labv1.LabSession{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(ls), got); err != nil {
		t.Fatalf("get LabSession: %v", err)
	}
	if !containsString(got.Finalizers, finalizer) {
		t.Fatalf("finalizer %q was not added", finalizer)
	}
}

func TestPauseReportsPausedOnlyAfterComputeStopsAndRetainsDisk(t *testing.T) {
	ctx := context.Background()
	s := controllerTestScheme(t)
	ls := &labv1.LabSession{
		ObjectMeta: metav1.ObjectMeta{Name: "session", Namespace: "uds-labs-vms", Finalizers: []string{finalizer}},
		Spec:       labv1.LabSessionSpec{Paused: true},
		Status:     labv1.LabSessionStatus{Phase: labv1.PhaseReady, ServiceDNS: "lab-session.uds-labs-vms.svc.cluster.local"},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithStatusSubresource(&labv1.LabSession{}).WithObjects(ls).Build()
	stopped := false
	p := testProvider{teardownCompute: func(context.Context, *labv1.LabSession) (bool, error) { return stopped, nil }}
	r := &LabSessionReconciler{Client: c, Scheme: s, Provider: p}
	req := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(ls)}

	result, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("pause while compute exists: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("pause must requeue while compute is still stopping")
	}
	got := &labv1.LabSession{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(ls), got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase == labv1.PhasePaused {
		t.Fatal("session reported Paused before compute stopped")
	}

	stopped = true
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatalf("finish pause: %v", err)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(ls), got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != labv1.PhasePaused || got.Status.ServiceDNS != "" {
		t.Fatalf("paused status = %+v", got.Status)
	}
}

func TestExpiryTearsDownRetainedDiskAndCompute(t *testing.T) {
	ctx := context.Background()
	s := controllerTestScheme(t)
	ls := &labv1.LabSession{
		ObjectMeta: metav1.ObjectMeta{Name: "expired", Namespace: "uds-labs-vms", Finalizers: []string{finalizer}},
		Spec:       labv1.LabSessionSpec{ExpiresAt: metav1.NewTime(metav1.Now().Add(-time.Minute))},
		Status:     labv1.LabSessionStatus{Phase: labv1.PhasePaused},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithStatusSubresource(&labv1.LabSession{}).WithObjects(ls).Build()
	teardownCalled := false
	p := testProvider{teardown: func(context.Context, *labv1.LabSession) error { teardownCalled = true; return nil }}
	r := &LabSessionReconciler{Client: c, Scheme: s, Provider: p}
	if _, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(ls)}); err != nil {
		t.Fatal(err)
	}
	if !teardownCalled {
		t.Fatal("expiry did not remove compute and retained disk")
	}
	got := &labv1.LabSession{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(ls), got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != labv1.PhaseExpired {
		t.Fatalf("phase=%s, want Expired", got.Status.Phase)
	}
}

func TestReconcileLifecycleStatusPreservesConcurrentStepUpdate(t *testing.T) {
	ctx := context.Background()
	s := controllerTestScheme(t)
	ls := &labv1.LabSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "session",
			Namespace:  "uds-labs-vms",
			Finalizers: []string{finalizer},
		},
	}
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithStatusSubresource(&labv1.LabSession{}).
		WithObjects(ls).
		Build()

	p := testProvider{reconcile: func(ctx context.Context, _ *labv1.LabSession) (provider.Result, error) {
		latest := &labv1.LabSession{}
		if err := c.Get(ctx, client.ObjectKeyFromObject(ls), latest); err != nil {
			return provider.Result{}, err
		}
		before := latest.DeepCopy()
		latest.Status.CompletedSteps = append(latest.Status.CompletedSteps, labv1.StepRecord{Step: "verify"})
		if err := c.Status().Patch(ctx, latest, client.MergeFrom(before)); err != nil {
			return provider.Result{}, err
		}
		return provider.Result{Phase: labv1.PhaseProvisioning, Message: "creating VM"}, nil
	}}
	r := &LabSessionReconciler{Client: c, Scheme: s, Provider: p}

	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{
		Name: ls.Name, Namespace: ls.Namespace,
	}})
	if err != nil {
		t.Fatalf("reconcile raced concurrent status writer: %v", err)
	}

	got := &labv1.LabSession{}
	if err := c.Get(ctx, client.ObjectKeyFromObject(ls), got); err != nil {
		t.Fatalf("get LabSession: %v", err)
	}
	if got.Status.Phase != labv1.PhaseProvisioning || got.Status.Message != "creating VM" {
		t.Fatalf("lifecycle status not applied: %+v", got.Status)
	}
	if len(got.Status.CompletedSteps) != 1 || got.Status.CompletedSteps[0].Step != "verify" {
		t.Fatalf("concurrent completed step was lost: %+v", got.Status.CompletedSteps)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
