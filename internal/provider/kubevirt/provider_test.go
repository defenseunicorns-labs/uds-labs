package kubevirt

import (
	"context"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kvv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
	"github.com/defenseunicorns-labs/uds-labs/internal/sizing"
)

// scenariosFS points at the real scenario fixtures relative to this package.
var scenariosFS = os.DirFS("../../../scenarios")

// ── goldenPVCForScenario ──────────────────────────────────────────────────────

func TestGoldenPVCForScenario_BaseFallback(t *testing.T) {
	p := New(Config{
		ScenariosFS: scenariosFS,
		GoldenPVCs:  map[string]string{"base": "golden-base"},
	})
	got, err := p.goldenPVCForScenario("playground-tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "golden-base" {
		t.Errorf("PVC name = %q, want golden-base", got)
	}
}

func TestGoldenPVCForScenario_PlaygroundToolsUsesBaseTier(t *testing.T) {
	p := New(Config{
		ScenariosFS: scenariosFS,
		GoldenPVCs:  map[string]string{"base": "golden-base"},
	})
	got, err := p.goldenPVCForScenario("playground-tools")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "golden-base" {
		t.Errorf("PVC name = %q, want golden-base", got)
	}
}

func TestGoldenPVCForScenario_PlaygroundUDSCoreTier(t *testing.T) {
	p := New(Config{
		ScenariosFS: scenariosFS,
		GoldenPVCs:  map[string]string{"uds-core": "golden-uds-core"},
	})
	got, err := p.goldenPVCForScenario("playground-uds-core")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "golden-uds-core" {
		t.Errorf("PVC name = %q, want golden-uds-core", got)
	}
}

func TestGoldenPVCForScenario_MissingTierReturnsError(t *testing.T) {
	p := New(Config{
		ScenariosFS: scenariosFS,
		GoldenPVCs:  map[string]string{},
	})
	_, err := p.goldenPVCForScenario("python-to-uds")
	if err == nil {
		t.Fatal("expected error for unconfigured tier")
	}
}

func TestGoldenPVCForScenario_UnknownScenarioReturnsError(t *testing.T) {
	p := New(Config{
		ScenariosFS: scenariosFS,
		GoldenPVCs:  map[string]string{"base": "golden-base"},
	})
	_, err := p.goldenPVCForScenario("does-not-exist")
	if err == nil {
		t.Fatal("expected error for unknown scenario")
	}
}

// ── ensureDataVolume ──────────────────────────────────────────────────────────

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatalf("add clientgo scheme: %v", err)
	}
	if err := cdiv1.AddToScheme(s); err != nil {
		t.Fatalf("add cdi scheme: %v", err)
	}
	if err := kvv1.AddToScheme(s); err != nil {
		t.Fatalf("add kubevirt scheme: %v", err)
	}
	if err := labv1.AddToScheme(s); err != nil {
		t.Fatalf("add labv1 scheme: %v", err)
	}
	return s
}

func testLabSession(name, namespace string) *labv1.LabSession {
	return &labv1.LabSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("test-uid-" + name),
		},
	}
}

func TestEnsureDataVolume_UsesPVCCloneSource(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("test-ls", "uds-labs-vms")

	p := New(Config{
		Client:             fc,
		Namespace:          "uds-labs-vms",
		GoldenPVCs:         map[string]string{"base": "golden-base"},
		GoldenPVCNamespace: "uds-labs-golden",
		GoldenPVCDiskSize:  "40Gi",
	})

	err := p.ensureDataVolume(context.Background(), ls, "lab-testdata", map[string]string{"test": "true"}, "golden-base")
	if err != nil {
		t.Fatalf("ensureDataVolume: %v", err)
	}

	dv := &cdiv1.DataVolume{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-testdata"}, dv); err != nil {
		t.Fatalf("get DataVolume: %v", err)
	}

	if dv.Spec.Source == nil || dv.Spec.Source.PVC == nil {
		t.Fatal("expected PVC clone source, got nil")
	}
	if dv.Spec.Source.Registry != nil {
		t.Error("registry source must be nil for PVC clone")
	}
	if dv.Spec.Source.PVC.Namespace != "uds-labs-golden" {
		t.Errorf("source PVC namespace = %q, want uds-labs-golden", dv.Spec.Source.PVC.Namespace)
	}
	if dv.Spec.Source.PVC.Name != "golden-base" {
		t.Errorf("source PVC name = %q, want golden-base", dv.Spec.Source.PVC.Name)
	}
	if dv.Spec.PVC != nil {
		t.Fatal("clone must use CDI storage API, not the direct PVC API")
	}
	if dv.Spec.Storage == nil {
		t.Fatal("clone storage spec is nil; CDI cannot apply filesystem overhead")
	}
	if dv.Spec.Storage.VolumeMode == nil || *dv.Spec.Storage.VolumeMode != corev1.PersistentVolumeFilesystem {
		t.Fatalf("clone volume mode = %v, want Filesystem for rootless CDI compatibility", dv.Spec.Storage.VolumeMode)
	}
	storage := dv.Spec.Storage.Resources.Requests["storage"]
	if storage.String() != "40Gi" {
		t.Errorf("storage = %q, want 40Gi", storage.String())
	}
}

func TestEnsureDataVolume_ResumeKeepsExistingSessionDisk(t *testing.T) {
	s := testScheme(t)
	ls := testLabSession("resume", "uds-labs-vms")
	existing := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "lab-resume", Namespace: "uds-labs-vms", CreationTimestamp: metav1.Now()},
		Spec:       cdiv1.DataVolumeSpec{Source: &cdiv1.DataVolumeSource{PVC: &cdiv1.DataVolumeSourcePVC{Namespace: "uds-labs-vms", Name: "original-golden"}}},
	}
	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	p := New(Config{Client: fc, Namespace: "uds-labs-vms"})
	if err := p.ensureDataVolume(context.Background(), ls, "lab-resume", nil, "different-golden"); err != nil {
		t.Fatal(err)
	}
	got := &cdiv1.DataVolume{}
	if err := fc.Get(context.Background(), client.ObjectKeyFromObject(existing), got); err != nil {
		t.Fatal(err)
	}
	if got.Spec.Source == nil || got.Spec.Source.PVC == nil || got.Spec.Source.PVC.Name != "original-golden" {
		t.Fatalf("resume replaced retained disk source: %+v", got.Spec.Source)
	}
}

func TestEnsureDataVolume_NamespaceFallsBackToVMNamespace(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("test-ls2", "uds-labs-vms")

	p := New(Config{
		Client:             fc,
		Namespace:          "uds-labs-vms",
		GoldenPVCs:         map[string]string{"base": "golden-base"},
		GoldenPVCNamespace: "", // empty → fall back to Namespace
		GoldenPVCDiskSize:  "40Gi",
	})

	if err := p.ensureDataVolume(context.Background(), ls, "lab-testdata2", nil, "golden-base"); err != nil {
		t.Fatalf("ensureDataVolume: %v", err)
	}

	dv := &cdiv1.DataVolume{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-testdata2"}, dv); err != nil {
		t.Fatalf("get DataVolume: %v", err)
	}

	if dv.Spec.Source.PVC.Namespace != "uds-labs-vms" {
		t.Errorf("source PVC namespace = %q, want uds-labs-vms (fallback)", dv.Spec.Source.PVC.Namespace)
	}
}

func TestEnsureDataVolume_DiskSizeFallsBackToDefault(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("test-ls3", "uds-labs-vms")

	p := New(Config{
		Client:            fc,
		Namespace:         "uds-labs-vms",
		GoldenPVCs:        map[string]string{"base": "golden-base"},
		GoldenPVCDiskSize: "", // empty → use defaultDiskSize
	})

	if err := p.ensureDataVolume(context.Background(), ls, "lab-testdata3", nil, "golden-base"); err != nil {
		t.Fatalf("ensureDataVolume: %v", err)
	}

	dv := &cdiv1.DataVolume{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-testdata3"}, dv); err != nil {
		t.Fatalf("get DataVolume: %v", err)
	}

	if dv.Spec.PVC != nil {
		t.Fatal("clone must use CDI storage API, not the direct PVC API")
	}
	if dv.Spec.Storage == nil {
		t.Fatal("clone storage spec is nil")
	}
	storage := dv.Spec.Storage.Resources.Requests["storage"]
	if storage.String() != defaultDiskSize {
		t.Errorf("storage = %q, want %q", storage.String(), defaultDiskSize)
	}
}

func TestEnsureDataVolume_PropagatesStorageClass(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("storage-class", "uds-labs-vms")
	p := New(Config{Client: fc, Namespace: "uds-labs-vms", StorageClass: "managed-csi-premium"})
	if err := p.ensureDataVolume(context.Background(), ls, "lab-storage", nil, "golden-base"); err != nil {
		t.Fatal(err)
	}
	dv := &cdiv1.DataVolume{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-storage"}, dv); err != nil {
		t.Fatal(err)
	}
	if dv.Spec.Storage == nil || dv.Spec.Storage.StorageClassName == nil || *dv.Spec.Storage.StorageClassName != "managed-csi-premium" {
		t.Fatalf("storage class not propagated: %+v", dv.Spec.Storage)
	}
}

func TestEnsureCapacityBootstrapCreatesAutoscalerCompatiblePod(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("capacity", "uds-labs-vms")
	p := New(Config{
		Client:       fc,
		Namespace:    "uds-labs-vms",
		NodeSelector: map[string]string{"kubernetes.azure.com/agentpool": "isv"},
		Tolerations:  []corev1.Toleration{{Key: "workload", Value: "isv", Effect: corev1.TaintEffectNoSchedule}},
	})

	ready, message, err := p.ensureCapacityBootstrap(context.Background(), ls, "lab-capacity", map[string]string{sessionLabel: "capacity"}, sizing.Spec{CPU: "4", Memory: "8Gi"})
	if err != nil || ready || message == "" {
		t.Fatalf("initial capacity bootstrap = ready:%t message:%q err:%v, want pending", ready, message, err)
	}

	pod := &corev1.Pod{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-capacity" + capacityBootstrapSuffix}, pod); err != nil {
		t.Fatalf("get capacity bootstrap Pod: %v", err)
	}
	if pod.Spec.NodeSelector["kubernetes.azure.com/agentpool"] != "isv" || len(pod.Spec.Tolerations) != 1 {
		t.Fatalf("bootstrap placement = selector:%v tolerations:%v", pod.Spec.NodeSelector, pod.Spec.Tolerations)
	}
	if pod.Labels[workloadLabel] != workloadLabelVMI || pod.Spec.Affinity == nil || len(pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 1 {
		t.Fatalf("bootstrap must reserve a distinct Lab node: labels=%v affinity=%+v", pod.Labels, pod.Spec.Affinity)
	}
	requests := pod.Spec.Containers[0].Resources.Requests
	cpu := requests[corev1.ResourceCPU]
	if got := cpu.String(); got != "4" {
		t.Errorf("bootstrap CPU request = %q, want 4", got)
	}
	memory := requests[corev1.ResourceMemory]
	if got := memory.String(); got != "8704Mi" {
		t.Errorf("bootstrap memory request = %q, want 8704Mi including KubeVirt overhead", got)
	}
	for _, device := range []corev1.ResourceName{"devices.kubevirt.io/kvm", "devices.kubevirt.io/tun", "devices.kubevirt.io/vhost-net"} {
		if _, found := requests[device]; found {
			t.Errorf("bootstrap must not request %q; that prevents cluster scale-from-zero", device)
		}
	}
}

func TestEnsureCapacityBootstrapReleasesKubeVirtReadyNode(t *testing.T) {
	s := testScheme(t)
	ls := testLabSession("ready-capacity", "uds-labs-vms")
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "lab-ready" + capacityBootstrapSuffix, Namespace: "uds-labs-vms", CreationTimestamp: metav1.Now()},
		Spec:       corev1.PodSpec{NodeName: "isv-0"},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "isv-0", Labels: map[string]string{
			"kubernetes.io/os":        "linux",
			"kubevirt.io/schedulable": "true",
		}},
		Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
			"devices.kubevirt.io/kvm":       resource.MustParse("1"),
			"devices.kubevirt.io/tun":       resource.MustParse("1"),
			"devices.kubevirt.io/vhost-net": resource.MustParse("1"),
		}},
	}
	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ls, pod, node).Build()
	p := New(Config{Client: fc, Namespace: "uds-labs-vms"})

	ready, message, err := p.ensureCapacityBootstrap(context.Background(), ls, "lab-ready", map[string]string{sessionLabel: "ready-capacity"}, sizing.Spec{CPU: "4", Memory: "8Gi"})
	if err != nil || !ready || message != "" {
		t.Fatalf("ready capacity bootstrap = ready:%t message:%q err:%v, want ready", ready, message, err)
	}
	if ls.Annotations[capacityReadyAnnotation] != "true" {
		t.Fatalf("capacity-ready annotation = %q, want true", ls.Annotations[capacityReadyAnnotation])
	}
	if err := fc.Get(context.Background(), client.ObjectKeyFromObject(pod), &corev1.Pod{}); !apierrors.IsNotFound(err) {
		t.Fatalf("bootstrap Pod still exists after capacity became ready: %v", err)
	}
}

func TestEnsureVMI_OptsLauncherOutOfAmbientMesh(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("test-vmi", "uds-labs-vms")
	p := New(Config{Client: fc, Namespace: "uds-labs-vms"})

	if err := p.ensureVMI(
		context.Background(),
		ls,
		"lab-test-vmi",
		map[string]string{sessionLabel: "test-vmi"},
		sizing.Spec{CPU: "1", Memory: "1Gi"},
	); err != nil {
		t.Fatalf("ensureVMI: %v", err)
	}

	vmi := &kvv1.VirtualMachineInstance{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-test-vmi"}, vmi); err != nil {
		t.Fatalf("get VMI: %v", err)
	}
	if got := vmi.Labels["istio.io/dataplane-mode"]; got != "none" {
		t.Fatalf("VMI ambient opt-out label = %q, want none", got)
	}
}

func TestEnsureVMI_RendersMasqueradePlacementAndOneLabPerNode(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("placement", "uds-labs-vms")
	p := New(Config{
		Client:       fc,
		Namespace:    "uds-labs-vms",
		NodeSelector: map[string]string{"labs.uds.dev/compute": "true"},
		Tolerations:  []corev1.Toleration{{Key: "workload", Value: "uds-labs", Effect: corev1.TaintEffectNoSchedule}},
	})
	if err := p.ensureVMI(context.Background(), ls, "lab-placement", map[string]string{sessionLabel: "placement"}, sizing.Spec{CPU: "8", Memory: "16Gi"}); err != nil {
		t.Fatal(err)
	}
	vmi := &kvv1.VirtualMachineInstance{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-placement"}, vmi); err != nil {
		t.Fatal(err)
	}
	if vmi.Spec.NodeSelector["labs.uds.dev/compute"] != "true" || len(vmi.Spec.Tolerations) != 1 {
		t.Fatalf("placement not rendered: selector=%v tolerations=%v", vmi.Spec.NodeSelector, vmi.Spec.Tolerations)
	}
	if len(vmi.Spec.Networks) != 1 || vmi.Spec.Networks[0].Pod == nil || len(vmi.Spec.Domain.Devices.Interfaces) != 1 || vmi.Spec.Domain.Devices.Interfaces[0].Masquerade == nil {
		t.Fatalf("explicit masquerade pod network missing: networks=%+v interfaces=%+v", vmi.Spec.Networks, vmi.Spec.Domain.Devices.Interfaces)
	}
	terms := vmi.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(terms) != 1 || terms[0].TopologyKey != corev1.LabelHostname || terms[0].LabelSelector.MatchLabels[workloadLabel] != workloadLabelVMI {
		t.Fatalf("one-Lab-per-node anti-affinity missing: %+v", terms)
	}
}

func TestEnsureNetworkPolicy_AllowsDNSAndExcludesInternalCIDRs(t *testing.T) {
	s := testScheme(t)
	fc := fake.NewClientBuilder().WithScheme(s).Build()
	ls := testLabSession("network", "uds-labs-vms")
	blocked := []string{"10.0.0.0/8", "169.254.0.0/16"}
	p := New(Config{Client: fc, Namespace: "uds-labs-vms", ServerNamespace: "uds-labs", BlockedEgressCIDRs: blocked})
	if err := p.ensureNetworkPolicy(context.Background(), ls, "lab-network", map[string]string{sessionLabel: "network"}); err != nil {
		t.Fatal(err)
	}
	np := &netv1.NetworkPolicy{}
	if err := fc.Get(context.Background(), client.ObjectKey{Namespace: "uds-labs-vms", Name: "lab-network"}, np); err != nil {
		t.Fatal(err)
	}
	if len(np.Spec.Egress) != 2 || len(np.Spec.Egress[0].To) != 0 || len(np.Spec.Egress[0].Ports) != 2 {
		t.Fatalf("portable TCP/UDP DNS rule missing: %+v", np.Spec.Egress)
	}
	internet := np.Spec.Egress[1].To[0].IPBlock
	if internet == nil || internet.CIDR != "0.0.0.0/0" || len(internet.Except) != 2 || internet.Except[0] != blocked[0] {
		t.Fatalf("internet exclusions not rendered: %+v", internet)
	}
}

func TestTeardownComputeRetainsSessionDataVolume(t *testing.T) {
	s := testScheme(t)
	ls := testLabSession("retain", "uds-labs-vms")
	ls.Annotations = map[string]string{capacityReadyAnnotation: "true"}
	name := resourceName(ls)
	dv := &cdiv1.DataVolume{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "uds-labs-vms"}}
	vmi := &kvv1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "uds-labs-vms"}}
	fc := fake.NewClientBuilder().WithScheme(s).WithObjects(ls, dv, vmi).Build()
	p := New(Config{Client: fc, Namespace: "uds-labs-vms"})
	stopped, err := p.TeardownCompute(context.Background(), ls)
	if err != nil || !stopped {
		t.Fatalf("teardown compute: stopped=%v err=%v", stopped, err)
	}
	if err := fc.Get(context.Background(), client.ObjectKeyFromObject(dv), &cdiv1.DataVolume{}); err != nil {
		t.Fatalf("pause deleted retained DataVolume: %v", err)
	}
	gotSession := &labv1.LabSession{}
	if err := fc.Get(context.Background(), client.ObjectKeyFromObject(ls), gotSession); err != nil {
		t.Fatal(err)
	}
	if gotSession.Annotations[capacityReadyAnnotation] != "" {
		t.Fatalf("pause retained capacity-ready state: %v", gotSession.Annotations)
	}
}
