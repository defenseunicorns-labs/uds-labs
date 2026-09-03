// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package labplatform_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func helmTemplate(t *testing.T, extraArgs ...string) string {
	t.Helper()
	args := append([]string{"template", "uds-labs", "./chart"}, extraArgs...)
	out, err := exec.Command("helm", args...).Output()
	if err != nil {
		t.Fatalf("helm template: %v\nstderr: %s", err, extractStderr(err))
	}
	return string(out)
}

func helmTemplateCommand(extraArgs ...string) *exec.Cmd {
	args := append([]string{"template", "uds-labs", "./chart"}, extraArgs...)
	return exec.Command("helm", args...)
}

func extractStderr(err error) []byte {
	var ee *exec.ExitError
	if exitErr, ok := err.(*exec.ExitError); ok {
		ee = exitErr
	}
	if ee != nil {
		return ee.Stderr
	}
	return nil
}

func TestHelmChart_OperatorConfigMapRendered(t *testing.T) {
	out := helmTemplate(t,
		"--set", "goldenPVCs.base=golden-base",
		"--set", "goldenPVCs.uds-core=golden-uds-core",
	)
	if !strings.Contains(out, "kind: ConfigMap") {
		t.Error("helm output missing ConfigMap")
	}
	if !strings.Contains(out, "golden-base") {
		t.Error("helm output missing golden-base PVC name")
	}
	if !strings.Contains(out, "golden-uds-core") {
		t.Error("helm output missing golden-uds-core PVC name")
	}
	if !strings.Contains(out, `goldenPVCDiskSize: "40Gi"`) {
		t.Error("default Session disk must be at least as large as the 40Gi UDS Core golden source")
	}
}

func TestHelmChart_OperatorConfigMapHasGoldenPVCNamespace(t *testing.T) {
	out := helmTemplate(t,
		"--set", "goldenPVCNamespace=uds-labs-golden",
		"--set", "goldenPVCs.base=golden-base",
	)
	if !strings.Contains(out, "uds-labs-golden") {
		t.Error("helm output missing goldenPVCNamespace")
	}
}

func TestHelmChart_OperatorDeploymentHasOperatorConfigEnv(t *testing.T) {
	out := helmTemplate(t)
	if !strings.Contains(out, "OPERATOR_CONFIG") {
		t.Error("operator deployment missing OPERATOR_CONFIG env var")
	}
}

func TestHelmChart_RendersPilotStorageCapacityPlacementAndAuthorization(t *testing.T) {
	out := helmTemplate(t,
		"--set", "storageClass=managed-csi-premium",
		"--set", "maxActiveSessions=1",
		"--set", "serverPlacement.nodeSelector.kubernetes\\.azure\\.com/agentpool=isv",
		"--set", "operatorPlacement.nodeSelector.kubernetes\\.azure\\.com/agentpool=isv",
		"--set-string", "vmiPlacement.nodeSelector.labs\\.uds\\.dev/compute=true",
		"--set", "auth.allowedGroups[0]=/UDS Labs/Pilot",
	)
	for _, expected := range []string{"storageClass: \"managed-csi-premium\"", "MAX_ACTIVE_SESSIONS", "labs.uds.dev/compute", "groups:", "anyOf:", "/UDS Labs/Pilot"} {
		if !strings.Contains(out, expected) {
			t.Errorf("helm output missing %q", expected)
		}
	}
	if strings.Contains(out, "volumesnapshots") || strings.Contains(out, "VolumeSnapshot") {
		t.Fatal("pilot chart must not render snapshot dependencies")
	}
}

func TestHelmChart_OperatorDeploymentHasConfigMapVolumeMount(t *testing.T) {
	out := helmTemplate(t)
	if !bytes.Contains([]byte(out), []byte("lab-operator-config")) {
		t.Error("operator deployment missing lab-operator-config volume/mount")
	}
}

func TestHelmChart_OperatorRBACCoversVMI(t *testing.T) {
	out := helmTemplate(t)
	if !strings.Contains(out, "virtualmachineinstances") {
		t.Error("operator Role missing virtualmachineinstances permission")
	}
}

func TestHelmChart_OperatorRBACCoversDataVolumes(t *testing.T) {
	out := helmTemplate(t)
	if !strings.Contains(out, "datavolumes") {
		t.Error("operator Role missing datavolumes permission")
	}
}

func TestHelmChart_OperatorRBACCoversLabSessionStatus(t *testing.T) {
	out := helmTemplate(t)
	if !strings.Contains(out, "labsessions/status") {
		t.Error("operator Role missing labsessions/status permission")
	}
}

func TestHelmChart_UsesPackagedImageByDefault(t *testing.T) {
	out := helmTemplate(t)
	var values chartValues
	readYAML(t, "chart/values.yaml", &values)

	if got := strings.Count(out, "image: "+values.Image); got != 2 {
		t.Fatalf("expected packaged image in both Deployments, got %d occurrences", got)
	}
	if strings.Contains(out, ":latest") {
		t.Fatal("rendered chart must not use a mutable latest tag")
	}
	if got := strings.Count(out, "imagePullPolicy: Always"); got != 2 {
		t.Fatalf("expected Always in both Deployments so same-version Zarf redeploys pull rebuilt images, got %d occurrences", got)
	}
}

func TestHelmChart_KubeVirtProviderExemptsCDIClonePodsFromIstioOverridePolicy(t *testing.T) {
	out := helmTemplate(t)
	for _, expected := range []string{
		"kind: Exemption",
		"namespace: uds-policy-exemptions",
		"RestrictIstioTrafficOverrides",
		"name: ^(cdi-(upload|clone-source)-.*|importer-.*|[0-9a-f-]{36}-source-pod)$",
		"namespace: uds-labs-vms",
	} {
		if !strings.Contains(out, expected) {
			t.Errorf("CDI clone pod exemption missing %q", expected)
		}
	}
}

func TestHelmChart_OperatorCanBeDisabledForPackageOnlyTests(t *testing.T) {
	// Zarf chart variables arrive as strings; preserve that package-level path here.
	out := helmTemplate(t, "--set-string", "operator.enabled=false")
	if strings.Contains(out, "name: lab-operator") {
		t.Fatal("disabled operator unexpectedly rendered the operator Deployment")
	}
	if strings.Contains(out, "name: lab-operator-config") {
		t.Fatal("disabled operator unexpectedly rendered the operator ConfigMap")
	}
	if !strings.Contains(out, "kind: Deployment\nmetadata:\n  name: uds-labs") {
		t.Fatal("disabling the operator must retain the application Deployment")
	}
}

func TestHelmChart_RejectsRemovedProviderValue(t *testing.T) {
	cmd := helmTemplateCommand("--set", "provider=pod")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected removed provider value to fail Helm validation\n%s", out)
	}
	if !strings.Contains(string(out), "provider") {
		t.Fatalf("expected provider validation error, got:\n%s", out)
	}
}
