// Package kubevirt implements provider.Provider on top of KubeVirt VMIs and CDI
// DataVolumes (ADR-0010/0012). One LabSession reconciles to: a DataVolume cloned
// from the scenario's OCI image, a VirtualMachineInstance, a headless Service
// exposing the in-VM ports, and a NetworkPolicy.
//
// SPIKE-DEPENDENT (see plan Phase 0 / Decisions D1–D3): the DataVolume strategy
// (clone-per-session vs golden-PVC vs containerDisk), the storage class/access
// mode, image digests, and confirmation that nested k3d runs inside the VMI are
// all unvalidated until the cluster spike runs. The shapes below are the
// intended design; constants are marked TODO where the spike will set them.
package kubevirt

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"text/template" // nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template -- renders cloud-init YAML/shell, not HTML

	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kvv1 "kubevirt.io/api/core/v1"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
	"github.com/defenseunicorns-labs/uds-labs/internal/cloudinit"
	"github.com/defenseunicorns-labs/uds-labs/internal/provider"
	"github.com/defenseunicorns-labs/uds-labs/internal/scenario"
	"github.com/defenseunicorns-labs/uds-labs/internal/sizing"
)

const (
	// sessionLabel is set on the VMI (and inherited by the virt-launcher pod) so
	// the Service and NetworkPolicy can select it.
	sessionLabel = "labs.uds.dev/session"
	// KubeVirt launcher pods must stay out of Istio ambient capture; ztunnel
	// cannot forward intercepted Service traffic through KubeVirt masquerade to
	// the guest VM. Pod-level intent overrides the ambient namespace label.
	istioDataplaneModeLabel = "istio.io/dataplane-mode"
	istioDataplaneModeNone  = "none"
	workloadLabel           = "labs.uds.dev/workload"
	workloadLabelVMI        = "vmi"

	// In-VM ports exposed by the lab VM software.
	portInject = 7680 // lab-inject.py (cmd/verify/navigate/services)
	portTTYD   = 7681 // ttyd main (tmux)
	portShell  = 7682 // ttyd direct bash
	portVNC    = 6080 // noVNC/websockify

	// defaultDiskSize is the fallback clone PVC size when GoldenPVCDiskSize is empty.
	defaultDiskSize = "40Gi"

	// capacityBootstrapSuffix distinguishes the temporary, ordinary Pod that
	// triggers cluster autoscaling before a VMI requires KubeVirt device resources.
	capacityBootstrapSuffix = "-capacity"
	capacityReadyAnnotation = "labs.uds.dev/capacity-bootstrap-ready"
	capacityBootstrapImage  = "registry.k8s.io/pause:3.10"
)

// Config wires the provider to the cluster and to scenario content.
type Config struct {
	Client client.Client
	// Namespace holds all VMIs/Services/NetworkPolicies (uds-labs-vms).
	Namespace string
	// ServerNamespace is allowed ingress to the VM Services (the platform server).
	ServerNamespace string

	// UserDataTmpl + ScenariosFS + InjectPy let the operator render cloud-init
	// itself (Decision D1: operator owns scenario content).
	UserDataTmpl *template.Template
	ScenariosFS  fs.FS
	InjectPy     string

	// SizeOverrides come from the operator's ConfigMap (ADR-0013); empty means
	// use sizing.Defaults.
	SizeOverrides map[sizing.Size]sizing.Spec

	// GoldenPVCs maps image tier (base|tools|uds-core) to a golden PVC name.
	// CDI clones one golden PVC per session instead of importing from a registry.
	GoldenPVCs         map[string]string
	GoldenPVCNamespace string // namespace where golden PVCs live; defaults to Namespace
	GoldenPVCDiskSize  string // usable disk size requested through CDI's storage API

	// StorageClass for the cloned DataVolume PVC. Empty uses the cluster default.
	StorageClass string

	// Placement applies portable scheduling constraints to virt-launcher Pods.
	NodeSelector map[string]string
	Affinity     *corev1.Affinity
	Tolerations  []corev1.Toleration

	// BlockedEgressCIDRs are excluded from otherwise permitted internet egress.
	// Cluster DNS is allowed separately.
	BlockedEgressCIDRs []string
}

var _ provider.Provider = (*Provider)(nil)

// Provider reconciles LabSessions into KubeVirt objects.
type Provider struct {
	cfg Config
}

// New builds a Provider.
func New(cfg Config) *Provider {
	return &Provider{cfg: cfg}
}

// resourceName derives the shared name for a session's child objects.
func resourceName(ls *labv1.LabSession) string {
	id := ls.Spec.SessionID
	if len(id) > 8 {
		id = id[:8]
	}
	return "lab-" + id
}

// serviceDNS is the in-cluster DNS the server proxies to.
func (p *Provider) serviceDNS(name string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", name, p.cfg.Namespace)
}

// Reconcile ensures the DataVolume, VMI, Service, and NetworkPolicy exist and
// reports progress. It is idempotent.
func (p *Provider) Reconcile(ctx context.Context, ls *labv1.LabSession) (provider.Result, error) {
	name := resourceName(ls)
	labels := map[string]string{sessionLabel: ls.Spec.SessionID}

	// Resolve golden PVC name + size + cloud-init from scenario content.
	goldenPVCName, err := p.goldenPVCForScenario(ls.Spec.ScenarioID)
	if err != nil {
		return provider.Result{Phase: labv1.PhaseFailed, Message: err.Error()}, nil
	}

	size, err := sizing.Normalize(sizing.Size(ls.Spec.Size))
	if err != nil {
		return provider.Result{Phase: labv1.PhaseFailed, Message: err.Error()}, nil
	}
	spec, ok := sizing.Resolve(size, p.cfg.SizeOverrides)
	if !ok {
		return provider.Result{Phase: labv1.PhaseFailed, Message: fmt.Sprintf("no resource spec for size %q", size)}, nil
	}

	userData, err := cloudinit.Render(p.cfg.UserDataTmpl, p.cfg.ScenariosFS, ls.Spec.ScenarioID, p.cfg.InjectPy, ls.Spec.BrowserEnabled)
	if err != nil {
		return provider.Result{Phase: labv1.PhaseFailed, Message: err.Error()}, nil
	}

	capacityReady, message, err := p.ensureCapacityBootstrap(ctx, ls, name, labels, spec)
	if err != nil {
		return provider.Result{}, fmt.Errorf("ensure capacity bootstrap: %w", err)
	}
	if !capacityReady {
		return provider.Result{Phase: labv1.PhaseProvisioning, Message: message}, nil
	}

	if err := p.ensureDataVolume(ctx, ls, name, labels, goldenPVCName); err != nil {
		return provider.Result{}, fmt.Errorf("ensure datavolume: %w", err)
	}
	if err := p.ensureUserDataSecret(ctx, ls, name, labels, userData); err != nil {
		return provider.Result{}, fmt.Errorf("ensure userdata secret: %w", err)
	}
	if err := p.ensureVMI(ctx, ls, name, labels, spec); err != nil {
		return provider.Result{}, fmt.Errorf("ensure vmi: %w", err)
	}
	if err := p.ensureService(ctx, ls, name, labels); err != nil {
		return provider.Result{}, fmt.Errorf("ensure service: %w", err)
	}
	if err := p.ensureNetworkPolicy(ctx, ls, name, labels); err != nil {
		return provider.Result{}, fmt.Errorf("ensure networkpolicy: %w", err)
	}

	// Read VMI phase. The operator promotes Running -> Ready only after the
	// ttyd HTTP probe passes (two-phase readiness, ADR-0011).
	vmi := &kvv1.VirtualMachineInstance{}
	if err := p.cfg.Client.Get(ctx, client.ObjectKey{Namespace: p.cfg.Namespace, Name: name}, vmi); err != nil {
		if apierrors.IsNotFound(err) {
			return provider.Result{Phase: labv1.PhaseProvisioning}, nil
		}
		return provider.Result{}, fmt.Errorf("get vmi: %w", err)
	}

	switch vmi.Status.Phase {
	case kvv1.Running:
		return provider.Result{Phase: labv1.PhaseRunning, ServiceDNS: p.serviceDNS(name)}, nil
	case kvv1.Failed, kvv1.Succeeded:
		return provider.Result{Phase: labv1.PhaseFailed, Message: fmt.Sprintf("vmi phase %s", vmi.Status.Phase)}, nil
	default:
		return provider.Result{Phase: labv1.PhaseProvisioning}, nil
	}
}

// TeardownCompute deletes the VMI, Service, and NetworkPolicy but leaves the
// DataVolume/PVC intact. It reports stopped only after the VMI is absent.
func (p *Provider) TeardownCompute(ctx context.Context, ls *labv1.LabSession) (bool, error) {
	name := resourceName(ls)
	objs := []client.Object{
		&kvv1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Namespace: p.cfg.Namespace, Name: name}},
		&netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Namespace: p.cfg.Namespace, Name: name}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: p.cfg.Namespace, Name: name}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: p.cfg.Namespace, Name: name + capacityBootstrapSuffix}},
	}
	for _, o := range objs {
		if err := p.cfg.Client.Delete(ctx, o); err != nil && !apierrors.IsNotFound(err) {
			return false, fmt.Errorf("delete %T %s: %w", o, name, err)
		}
	}
	if err := p.clearCapacityBootstrap(ctx, ls); err != nil {
		return false, fmt.Errorf("clear capacity bootstrap state: %w", err)
	}

	vmi := &kvv1.VirtualMachineInstance{}
	err := p.cfg.Client.Get(ctx, client.ObjectKey{Namespace: p.cfg.Namespace, Name: name}, vmi)
	if apierrors.IsNotFound(err) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("confirm vmi %s stopped: %w", name, err)
	}
	return false, nil
}

// teardownDisk deletes the DataVolume (and therefore its underlying PVC).
func (p *Provider) teardownDisk(ctx context.Context, ls *labv1.LabSession) error {
	name := resourceName(ls)
	dv := &cdiv1.DataVolume{ObjectMeta: metav1.ObjectMeta{Namespace: p.cfg.Namespace, Name: name}}
	if err := p.cfg.Client.Delete(ctx, dv); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete datavolume %s: %w", name, err)
	}
	return nil
}

// Teardown deletes all session objects, including the retained disk.
func (p *Provider) Teardown(ctx context.Context, ls *labv1.LabSession) error {
	if _, err := p.TeardownCompute(ctx, ls); err != nil {
		return err
	}
	return p.teardownDisk(ctx, ls)
}

// goldenPVCForScenario maps a scenario to its golden PVC name.
// Tier resolution: explicit sc.Image override → playground-<tier> prefix → "base".
// The tools playground uses the consolidated base image.
func (p *Provider) goldenPVCForScenario(scenarioID string) (string, error) {
	sc, err := scenario.Load(p.cfg.ScenariosFS, scenarioID)
	if err != nil {
		return "", fmt.Errorf("load scenario %q: %w", scenarioID, err)
	}

	tier := "base"
	switch {
	case sc.Image != "":
		tier = sc.Image
	case sc.Playground:
		const prefix = "playground-"
		if len(scenarioID) > len(prefix) && scenarioID[:len(prefix)] == prefix {
			tier = scenarioID[len(prefix):]
		}
	}
	if tier == "tools" {
		tier = "base"
	}

	pvcName, ok := p.cfg.GoldenPVCs[tier]
	if !ok || pvcName == "" {
		return "", fmt.Errorf("no golden PVC configured for tier %q (scenario %q)", tier, scenarioID)
	}
	return pvcName, nil
}

func (p *Provider) ensureDataVolume(ctx context.Context, ls *labv1.LabSession, name string, labels map[string]string, goldenPVCName string) error {
	diskSizeStr := p.cfg.GoldenPVCDiskSize
	if diskSizeStr == "" {
		diskSizeStr = defaultDiskSize
	}
	diskQ, err := resource.ParseQuantity(diskSizeStr)
	if err != nil {
		return err
	}

	srcNamespace := p.cfg.GoldenPVCNamespace
	if srcNamespace == "" {
		srcNamespace = p.cfg.Namespace
	}

	// Fresh sessions clone the golden PVC. On resume CreateOrUpdate observes the
	// existing DataVolume and leaves its immutable source and storage unchanged.
	source := &cdiv1.DataVolumeSource{
		PVC: &cdiv1.DataVolumeSourcePVC{
			Namespace: srcNamespace,
			Name:      goldenPVCName,
		},
	}

	dv := &cdiv1.DataVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.cfg.Namespace, Labels: labels},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, p.cfg.Client, dv, func() error {
		dv.Labels = labels
		// DataVolume spec (source + storage shape) is immutable once CDI begins
		// importing. Only set it on create to avoid validation errors on update.
		if dv.CreationTimestamp.IsZero() {
			dv.Spec = cdiv1.DataVolumeSpec{
				Source: source,
				// Use CDI's storage API so its webhook applies the configured
				// filesystem overhead. The golden imports use the same API; using
				// the direct PVC API here would make an 40Gi clone smaller than
				// the overhead-inflated 40Gi source PVC.
				Storage: &cdiv1.StorageSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
					// Rootless CDI raw-block operations require CRI device ownership
					// support that managed AKS containerd does not provide.
					VolumeMode: ptr(corev1.PersistentVolumeFilesystem),
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{corev1.ResourceStorage: diskQ},
					},
					StorageClassName: storageClassPtr(p.cfg.StorageClass),
				},
			}
		}
		return controllerutil.SetControllerReference(ls, dv, p.cfg.Client.Scheme())
	})
	return err
}

func (p *Provider) ensureUserDataSecret(ctx context.Context, ls *labv1.LabSession, name string, labels map[string]string, userData string) error {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.cfg.Namespace, Labels: labels}}
	_, err := controllerutil.CreateOrUpdate(ctx, p.cfg.Client, secret, func() error {
		secret.StringData = map[string]string{"userdata": userData}
		return controllerutil.SetControllerReference(ls, secret, p.cfg.Client.Scheme())
	})
	return err
}

// ensureCapacityBootstrap creates an ordinary Pod with the Lab's placement and
// resource footprint before creating its VMI. Unlike a virt-launcher Pod, it
// does not request KubeVirt device resources, so a cluster autoscaler can use
// it to grow a zero-sized KubeVirt node pool. It is deleted once its node is
// KubeVirt-ready; the annotation prevents it from being recreated for this
// compute lifecycle.
func (p *Provider) ensureCapacityBootstrap(ctx context.Context, ls *labv1.LabSession, name string, labels map[string]string, spec sizing.Spec) (bool, string, error) {
	if ls.Annotations[capacityReadyAnnotation] == "true" {
		return true, "", nil
	}

	cpu, err := resource.ParseQuantity(spec.CPU)
	if err != nil {
		return false, "", fmt.Errorf("parse capacity CPU %q: %w", spec.CPU, err)
	}
	memory, err := resource.ParseQuantity(spec.Memory)
	if err != nil {
		return false, "", fmt.Errorf("parse capacity memory %q: %w", spec.Memory, err)
	}
	// KubeVirt adds launcher overhead beyond a VMI's requested guest memory.
	// Reserve a conservative amount so the bootstrap Pod cannot land on a node
	// that its eventual VMI would fail to fit.
	memory.Add(resource.MustParse("512Mi"))

	bootstrapLabels := make(map[string]string, len(labels)+2)
	for key, value := range labels {
		bootstrapLabels[key] = value
	}
	bootstrapLabels[workloadLabel] = workloadLabelVMI
	bootstrapLabels[istioDataplaneModeLabel] = istioDataplaneModeNone

	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name + capacityBootstrapSuffix, Namespace: p.cfg.Namespace}}
	_, err = controllerutil.CreateOrUpdate(ctx, p.cfg.Client, pod, func() error {
		pod.Labels = bootstrapLabels
		if pod.CreationTimestamp.IsZero() {
			pod.Spec = corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyAlways,
				NodeSelector:  copyStringMap(p.cfg.NodeSelector),
				Affinity:      withLabAntiAffinity(p.cfg.Affinity),
				Tolerations:   append([]corev1.Toleration(nil), p.cfg.Tolerations...),
				Containers: []corev1.Container{{
					Name:  "reserve-capacity",
					Image: capacityBootstrapImage,
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU:    cpu,
						corev1.ResourceMemory: memory,
					}},
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr(false),
						ReadOnlyRootFilesystem:   ptr(true),
						RunAsNonRoot:             ptr(true),
						Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
					},
				}},
			}
		}
		return controllerutil.SetControllerReference(ls, pod, p.cfg.Client.Scheme())
	})
	if err != nil {
		return false, "", err
	}

	if pod.Status.Phase != corev1.PodRunning || pod.Spec.NodeName == "" {
		return false, "Waiting for cluster capacity to schedule the Lab bootstrap Pod", nil
	}

	node := &corev1.Node{}
	if err := p.cfg.Client.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, node); err != nil {
		return false, "", fmt.Errorf("get bootstrap node %q: %w", pod.Spec.NodeName, err)
	}
	if node.Labels["kubernetes.io/os"] != "linux" {
		return false, fmt.Sprintf("Waiting for node %q to report kubernetes.io/os=linux", node.Name), nil
	}
	if node.Labels["kubevirt.io/schedulable"] != "true" {
		return false, fmt.Sprintf("Waiting for KubeVirt to make node %q schedulable", node.Name), nil
	}
	for _, device := range []corev1.ResourceName{"devices.kubevirt.io/kvm", "devices.kubevirt.io/tun", "devices.kubevirt.io/vhost-net"} {
		quantity, ok := node.Status.Allocatable[device]
		if !ok || quantity.Sign() <= 0 {
			return false, fmt.Sprintf("Waiting for KubeVirt device resource %q on node %q", device, node.Name), nil
		}
	}

	if err := p.cfg.Client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
		return false, "", fmt.Errorf("delete capacity bootstrap Pod: %w", err)
	}
	if err := p.markCapacityBootstrapReady(ctx, ls); err != nil {
		return false, "", err
	}
	return true, "", nil
}

func (p *Provider) markCapacityBootstrapReady(ctx context.Context, ls *labv1.LabSession) error {
	before := ls.DeepCopy()
	if ls.Annotations == nil {
		ls.Annotations = map[string]string{}
	}
	ls.Annotations[capacityReadyAnnotation] = "true"
	return p.cfg.Client.Patch(ctx, ls, client.MergeFrom(before))
}

func (p *Provider) clearCapacityBootstrap(ctx context.Context, ls *labv1.LabSession) error {
	if !ls.DeletionTimestamp.IsZero() || ls.Annotations[capacityReadyAnnotation] == "" {
		return nil
	}
	before := ls.DeepCopy()
	delete(ls.Annotations, capacityReadyAnnotation)
	return p.cfg.Client.Patch(ctx, ls, client.MergeFrom(before))
}

func (p *Provider) ensureVMI(ctx context.Context, ls *labv1.LabSession, name string, labels map[string]string, spec sizing.Spec) error {
	cpuQ, err := resource.ParseQuantity(spec.CPU)
	if err != nil {
		return fmt.Errorf("parse cpu %q: %w", spec.CPU, err)
	}
	memQ, err := resource.ParseQuantity(spec.Memory)
	if err != nil {
		return fmt.Errorf("parse memory %q: %w", spec.Memory, err)
	}

	vmiLabels := make(map[string]string, len(labels)+2)
	for key, value := range labels {
		vmiLabels[key] = value
	}
	vmiLabels[istioDataplaneModeLabel] = istioDataplaneModeNone
	vmiLabels[workloadLabel] = workloadLabelVMI

	vmi := &kvv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.cfg.Namespace, Labels: vmiLabels},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, p.cfg.Client, vmi, func() error {
		// VMI spec is effectively immutable once Running; only set on create.
		if vmi.CreationTimestamp.IsZero() {
			vmi.Labels = vmiLabels
			vmi.Spec = kvv1.VirtualMachineInstanceSpec{
				NodeSelector: copyStringMap(p.cfg.NodeSelector),
				Affinity:     withLabAntiAffinity(p.cfg.Affinity),
				Tolerations:  append([]corev1.Toleration(nil), p.cfg.Tolerations...),
				Domain: kvv1.DomainSpec{
					Resources: kvv1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    cpuQ,
							corev1.ResourceMemory: memQ,
						},
					},
					Devices: kvv1.Devices{
						Interfaces: []kvv1.Interface{{
							Name:                   "default",
							InterfaceBindingMethod: kvv1.InterfaceBindingMethod{Masquerade: &kvv1.InterfaceMasquerade{}},
						}},
						Disks: []kvv1.Disk{
							{Name: "rootdisk", DiskDevice: kvv1.DiskDevice{Disk: &kvv1.DiskTarget{Bus: kvv1.DiskBusVirtio}}},
							{Name: "cloudinitdisk", DiskDevice: kvv1.DiskDevice{Disk: &kvv1.DiskTarget{Bus: kvv1.DiskBusVirtio}}},
						},
					},
				},
				Networks: []kvv1.Network{{
					Name:          "default",
					NetworkSource: kvv1.NetworkSource{Pod: &kvv1.PodNetwork{}},
				}},
				Volumes: []kvv1.Volume{
					{Name: "rootdisk", VolumeSource: kvv1.VolumeSource{DataVolume: &kvv1.DataVolumeSource{Name: name}}},
					{Name: "cloudinitdisk", VolumeSource: kvv1.VolumeSource{CloudInitNoCloud: &kvv1.CloudInitNoCloudSource{UserDataSecretRef: &corev1.LocalObjectReference{Name: name}}}},
				},
			}
		}
		return controllerutil.SetControllerReference(ls, vmi, p.cfg.Client.Scheme())
	})
	return err
}

func (p *Provider) ensureService(ctx context.Context, ls *labv1.LabSession, name string, labels map[string]string) error {
	ports := []corev1.ServicePort{
		{Name: "inject", Port: portInject, TargetPort: intstr.FromInt(portInject)},
		{Name: "ttyd", Port: portTTYD, TargetPort: intstr.FromInt(portTTYD)},
		{Name: "shell", Port: portShell, TargetPort: intstr.FromInt(portShell)},
	}
	if ls.Spec.BrowserEnabled {
		ports = append(ports, corev1.ServicePort{Name: "vnc", Port: portVNC, TargetPort: intstr.FromInt(portVNC)})
	}

	svc := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.cfg.Namespace, Labels: labels}}
	_, err := controllerutil.CreateOrUpdate(ctx, p.cfg.Client, svc, func() error {
		svc.Labels = labels
		svc.Spec.ClusterIP = corev1.ClusterIPNone // headless: resolves to the launcher pod IP
		svc.Spec.Selector = labels
		svc.Spec.Ports = ports
		return controllerutil.SetControllerReference(ls, svc, p.cfg.Client.Scheme())
	})
	return err
}

func (p *Provider) ensureNetworkPolicy(ctx context.Context, ls *labv1.LabSession, name string, labels map[string]string) error {
	tcp := corev1.ProtocolTCP
	udp := corev1.ProtocolUDP
	dns := intstr.FromInt(53)

	ingressPorts := []netv1.NetworkPolicyPort{
		{Protocol: &tcp, Port: ptr(intstr.FromInt(portInject))},
		{Protocol: &tcp, Port: ptr(intstr.FromInt(portTTYD))},
		{Protocol: &tcp, Port: ptr(intstr.FromInt(portShell))},
	}
	if ls.Spec.BrowserEnabled {
		ingressPorts = append(ingressPorts, netv1.NetworkPolicyPort{Protocol: &tcp, Port: ptr(intstr.FromInt(portVNC))})
	}

	blocked := make([]string, 0, len(p.cfg.BlockedEgressCIDRs))
	for _, cidr := range p.cfg.BlockedEgressCIDRs {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid blocked egress CIDR %q: %w", cidr, err)
		}
		if ip.To4() == nil {
			return fmt.Errorf("blocked egress CIDR %q is IPv6; IPv6 internet egress is not enabled", cidr)
		}
		blocked = append(blocked, cidr)
	}

	np := &netv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: p.cfg.Namespace, Labels: labels}}
	_, err := controllerutil.CreateOrUpdate(ctx, p.cfg.Client, np, func() error {
		np.Labels = labels
		np.Spec = netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: labels},
			PolicyTypes: []netv1.PolicyType{netv1.PolicyTypeIngress, netv1.PolicyTypeEgress},
			// Ingress: only the platform server's namespace may reach the VM ports
			// (Phase E's proxy depends on this). VMI<->VMI is denied by omission.
			Ingress: []netv1.NetworkPolicyIngressRule{{
				From: []netv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"kubernetes.io/metadata.name": p.cfg.ServerNamespace},
					},
				}},
				Ports: ingressPorts,
			}},
			// Egress: cluster DNS plus public IPv4 internet, excluding configured
			// cluster, internal, link-local, metadata, and platform ranges.
			Egress: []netv1.NetworkPolicyEgressRule{
				{
					// Permit DNS to the resolver supplied to the guest by DHCP. A
					// namespace/pod selector is not portable through KubeVirt
					// masquerade, service DNAT, and every supported CNI implementation.
					Ports: []netv1.NetworkPolicyPort{
						{Protocol: &udp, Port: &dns},
						{Protocol: &tcp, Port: &dns},
					},
				},
				{To: []netv1.NetworkPolicyPeer{{IPBlock: &netv1.IPBlock{CIDR: "0.0.0.0/0", Except: blocked}}}},
			},
		}
		return controllerutil.SetControllerReference(ls, np, p.cfg.Client.Scheme())
	})
	return err
}

func copyStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func withLabAntiAffinity(configured *corev1.Affinity) *corev1.Affinity {
	var affinity *corev1.Affinity
	if configured == nil {
		affinity = &corev1.Affinity{}
	} else {
		affinity = configured.DeepCopy()
	}
	if affinity.PodAntiAffinity == nil {
		affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
	}
	affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
		affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		corev1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{MatchLabels: map[string]string{workloadLabel: workloadLabelVMI}},
			TopologyKey:   corev1.LabelHostname,
		},
	)
	return affinity
}

func storageClassPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptr[T any](v T) *T { return &v }
