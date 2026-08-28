// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package session

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
)

func sessionTestClient(t *testing.T, objects ...runtime.Object) *Manager {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := labv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	clientObjects := make([]runtime.Object, len(objects))
	copy(clientObjects, objects)
	c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(clientObjects...).Build()
	return NewManager(c, "uds-labs-vms", time.Hour, os.DirFS("../../scenarios"), 1)
}

func TestCreateSerializesConcurrentCapacityChecks(t *testing.T) {
	mgr := sessionTestClient(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, clientID := range []string{"client-a", "client-b"} {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			<-start
			_, err := mgr.Create(context.Background(), id, "python-to-uds", id+"@example.com")
			errs <- err
		}(clientID)
	}
	close(start)
	wg.Wait()
	close(errs)

	created, capacity := 0, 0
	for err := range errs {
		switch {
		case err == nil:
			created++
		case errors.Is(err, ErrCapacityReached):
			capacity++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if created != 1 || capacity != 1 {
		t.Fatalf("created=%d capacity-rejected=%d, want 1 each", created, capacity)
	}
}

func TestCreateCountsPausedAndRetainedFailedSessionsAgainstCapacity(t *testing.T) {
	for _, phase := range []labv1.LabSessionPhase{labv1.PhasePaused, labv1.PhaseFailed} {
		t.Run(string(phase), func(t *testing.T) {
			existing := &labv1.LabSession{
				ObjectMeta: metav1.ObjectMeta{Name: "existing", Namespace: "uds-labs-vms"},
				Spec: labv1.LabSessionSpec{
					SessionID: "existing", ClientID: "other", ExpiresAt: metav1.NewTime(time.Now().Add(time.Hour)),
				},
				Status: labv1.LabSessionStatus{Phase: phase},
			}
			mgr := sessionTestClient(t, existing)
			_, err := mgr.Create(context.Background(), "new-client", "python-to-uds", "new@example.com")
			if !errors.Is(err, ErrCapacityReached) {
				t.Fatalf("create error = %v, want capacity reached", err)
			}
		})
	}
}
