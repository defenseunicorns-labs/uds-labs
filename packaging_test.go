// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package labplatform_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
	"gopkg.in/yaml.v3"
)

type zarfPackage struct {
	Components []zarfComponent `yaml:"components"`
}

type zarfComponent struct {
	Name          string   `yaml:"name"`
	Images        []string `yaml:"images"`
	ImageArchives []struct {
		Path   string   `yaml:"path"`
		Images []string `yaml:"images"`
	} `yaml:"imageArchives"`
	Charts []struct {
		LocalPath string `yaml:"localPath"`
	} `yaml:"charts"`
	Actions struct {
		OnDeploy struct {
			After []struct {
				Cmd  string `yaml:"cmd"`
				Wait struct {
					Cluster struct {
						Kind      string `yaml:"kind"`
						Name      string `yaml:"name"`
						Namespace string `yaml:"namespace"`
						Condition string `yaml:"condition"`
					} `yaml:"cluster"`
				} `yaml:"wait"`
			} `yaml:"after"`
		} `yaml:"onDeploy"`
	} `yaml:"actions"`
}

type applicationPackage struct {
	Metadata struct {
		Version string `yaml:"version"`
	} `yaml:"metadata"`
	Variables []struct {
		Name string `yaml:"name"`
	} `yaml:"variables"`
	Components []struct {
		Name   string   `yaml:"name"`
		Images []string `yaml:"images"`
		Charts []struct {
			Version string `yaml:"version"`
		} `yaml:"charts"`
	} `yaml:"components"`
}

type chartValues struct {
	Image string `yaml:"image"`
}

type chartMetadata struct {
	Version    string `yaml:"version"`
	AppVersion string `yaml:"appVersion"`
}

type taskConfig struct {
	Tasks []taskDefinition `yaml:"tasks"`
}

type taskDefinition struct {
	Name    string       `yaml:"name"`
	Actions []taskAction `yaml:"actions"`
}

type taskAction struct {
	Task string `yaml:"task"`
	Cmd  string `yaml:"cmd"`
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func allTaskContents(t *testing.T) string {
	t.Helper()
	paths, err := filepath.Glob("tasks/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	paths = append([]string{"tasks.yaml"}, paths...)
	var contents strings.Builder
	for _, path := range paths {
		contents.Write(mustReadFile(t, path))
	}
	return contents.String()
}

func taskDefinitions(t *testing.T) map[string]*taskDefinition {
	t.Helper()
	paths, err := filepath.Glob("tasks/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	definitions := map[string]*taskDefinition{}
	for _, path := range paths {
		var config taskConfig
		readYAML(t, path, &config)
		for i := range config.Tasks {
			definitions[config.Tasks[i].Name] = &config.Tasks[i]
		}
	}
	return definitions
}

func TestLabSessionAPIGroupIsConsistent(t *testing.T) {
	var crd struct {
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec struct {
			Group string `yaml:"group"`
		} `yaml:"spec"`
	}
	readYAML(t, "deploy/crd/labs.uds.dev_labsessions.yaml", &crd)

	wantGroup := labv1.GroupVersion.Group
	if crd.Spec.Group != wantGroup {
		t.Fatalf("CRD group %q differs from Go API group %q", crd.Spec.Group, wantGroup)
	}
	if crd.Metadata.Name != "labsessions."+wantGroup {
		t.Fatalf("CRD name %q does not match API group %q", crd.Metadata.Name, wantGroup)
	}

	rendered := helmTemplate(t)
	if !strings.Contains(rendered, `apiGroups: ["`+wantGroup+`"]`) {
		t.Fatalf("rendered RBAC does not grant access to API group %q", wantGroup)
	}
	if strings.Contains(rendered, `apiGroups: ["lab.uds.dev"]`) {
		t.Fatal("rendered RBAC still grants access to the legacy singular API group")
	}
}

func TestLabCRDRemovalUnblocksFinalizedSessions(t *testing.T) {
	zarf := string(mustReadFile(t, "zarf.yaml"))
	for _, required := range []string{
		"onRemove:",
		"Remove orphaned LabSession teardown finalizers",
		"patch labsession \"$name\"",
		`-p '{"metadata":{"finalizers":[]}}'`,
	} {
		if !strings.Contains(zarf, required) {
			t.Errorf("lab-crds removal action is missing %q", required)
		}
	}
}

func TestVMImageComponentsInDedicatedPackage(t *testing.T) {
	var versionedPackage struct {
		Metadata struct {
			Version string `yaml:"version"`
		} `yaml:"metadata"`
		Components []struct {
			Charts []struct {
				Version string `yaml:"version"`
			} `yaml:"charts"`
		} `yaml:"components"`
	}
	readYAML(t, "packages/vm-images/zarf.yaml", &versionedPackage)
	if versionedPackage.Metadata.Version == "" {
		t.Fatal("VM image package version must be set")
	}
	for _, component := range versionedPackage.Components {
		for _, chart := range component.Charts {
			if chart.Version != versionedPackage.Metadata.Version {
				t.Fatalf("VM image chart version %q does not match package version %q", chart.Version, versionedPackage.Metadata.Version)
			}
		}
	}
	bundle := string(mustReadFile(t, "bundle/uds-bundle.yaml"))
	if !strings.Contains(bundle, "ref: "+versionedPackage.Metadata.Version) {
		t.Fatalf("bundle does not reference VM image package version %q", versionedPackage.Metadata.Version)
	}

	contents, err := os.ReadFile("packages/vm-images/zarf.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var pkg zarfPackage
	if err := yaml.Unmarshal(contents, &pkg); err != nil {
		t.Fatalf("parse package: %v", err)
	}

	var server, imports *zarfComponent
	serverIdx := -1
	for i := range pkg.Components {
		switch pkg.Components[i].Name {
		case "vm-image-server":
			server = &pkg.Components[i]
			serverIdx = i
		case "golden-pvcs":
			imports = &pkg.Components[i]
		}
	}
	if server == nil || imports == nil {
		t.Fatal("packages/vm-images/zarf.yaml must contain both vm-image-server and golden-pvcs components")
	}
	for i := range pkg.Components {
		if pkg.Components[i].Name == "golden-pvcs" && i <= serverIdx {
			t.Fatal("vm-image-server must come before golden-pvcs in component order")
		}
	}

	if len(server.Images) != 0 {
		t.Fatal("vm-image-server must use pre-exported imageArchives instead of re-ingesting large images from Docker")
	}
	if len(server.ImageArchives) != 2 {
		t.Fatalf("vm-image-server must have 2 image archives (base + uds-core), got %d", len(server.ImageArchives))
	}
	for index, expected := range []struct {
		path  string
		image string
	}{
		{path: "base.tar", image: "ghcr.io/defenseunicorns-labs/uds-labs-vm-images/base:0.4.1"},
		{path: "uds-core.tar", image: "ghcr.io/defenseunicorns-labs/uds-labs-vm-images/uds-core:0.4.1"},
	} {
		archive := server.ImageArchives[index]
		if archive.Path != expected.path || len(archive.Images) != 1 || archive.Images[0] != expected.image {
			t.Fatalf("unexpected VM image archive: %#v", archive)
		}
	}

	if len(server.Charts) != 1 || server.Charts[0].LocalPath != "chart" || len(imports.Charts) != 1 || imports.Charts[0].LocalPath != "chart" {
		t.Fatal("both VM image components must render the portable Helm chart")
	}
	if len(imports.Actions.OnDeploy.After) != 3 {
		t.Fatal("golden-pvcs component must wait for both imports before scaling down its servers")
	}
	for index, name := range []string{"golden-base", "golden-uds-core"} {
		wait := imports.Actions.OnDeploy.After[index].Wait.Cluster
		if wait.Kind != "datavolume" || wait.Name != name || wait.Namespace != "uds-labs-vms" || wait.Condition != "ready" {
			t.Fatalf("unexpected golden DataVolume readiness wait: %#v", wait)
		}
	}
	wantScale := "./zarf tools kubectl scale deployment/vm-image-base deployment/vm-image-uds-core --replicas=0 --namespace=uds-labs-vms"
	if got := imports.Actions.OnDeploy.After[2].Cmd; got != wantScale {
		t.Fatalf("image-server scale-down command = %q, want %q", got, wantScale)
	}
	if len(server.Actions.OnDeploy.After) != 2 {
		t.Fatal("vm-image-server component must wait for UDS networking and its server pods")
	}
	packageWait := server.Actions.OnDeploy.After[0].Wait.Cluster
	if packageWait.Kind != "package" || packageWait.Name != "uds-labs-vm-images" || packageWait.Namespace != "uds-labs-vms" || packageWait.Condition != "ready" {
		t.Fatalf("unexpected UDS Package readiness wait: %#v", packageWait)
	}
	podWait := server.Actions.OnDeploy.After[1].Wait.Cluster
	if podWait.Kind != "pod" || podWait.Name != "app.kubernetes.io/part-of=uds-labs-vm-images" || podWait.Namespace != "uds-labs-vms" || podWait.Condition != "ready" {
		t.Fatalf("unexpected image-server Pod readiness wait: %#v", podWait)
	}

	cmd := exec.Command("helm", "template", "vm-images", "packages/vm-images/chart")
	rendered, err := cmd.Output()
	if err != nil {
		t.Fatalf("render VM image chart: %v", err)
	}
	serverText := string(rendered)
	for _, policyFragment := range []string{
		"cdi.kubevirt.io: importer",
		"app.kubernetes.io/part-of: uds-labs-vm-images",
		"cdi.kubevirt.io: cdi-clone-source",
		"cdi.kubevirt.io: cdi-upload-server",
		"port: 8080",
		"port: 8443",
	} {
		if !strings.Contains(serverText, policyFragment) {
			t.Fatalf("rendered least-privilege UDS policy missing %q", policyFragment)
		}
	}
	if strings.Contains(serverText, "remoteGenerated: Anywhere") {
		t.Fatal("VM image package must not render an Anywhere network allowance")
	}
	for _, image := range []string{
		"ghcr.io/defenseunicorns-labs/uds-labs-vm-images/base:0.4.1",
		"ghcr.io/defenseunicorns-labs/uds-labs-vm-images/uds-core:0.4.1",
	} {
		if !strings.Contains(serverText, image) {
			t.Fatalf("image server chart does not reference packaged image %q", image)
		}
	}
	if count := strings.Count(serverText, `cdi.kubevirt.io/storage.bind.immediate.requested: "true"`); count != 2 {
		t.Fatalf("both golden DataVolumes must request immediate binding, got %d annotations", count)
	}
	if count := strings.Count(serverText, "helm.sh/resource-policy: keep"); count != 2 {
		t.Fatalf("both golden DataVolumes must be retained across upgrades, got %d keep annotations", count)
	}
	for _, forbidden := range []string{"source:\n    registry:", "secretRef:", "ZARF_REGISTRY", "CDI_REGISTRY", "zarf-docker-registry"} {
		if strings.Contains(serverText, forbidden) {
			t.Fatalf("DataVolumes must not contain registry wiring %q", forbidden)
		}
	}
	for _, url := range []string{
		"http://vm-image-base.uds-labs-vms.svc.cluster.local:8080/lab-base.qcow2",
		"http://vm-image-uds-core.uds-labs-vms.svc.cluster.local:8080/lab-playground-uds-core.qcow2",
	} {
		if !strings.Contains(serverText, url) {
			t.Fatalf("DataVolume chart does not contain image-server URL %q", url)
		}
	}

	for _, dockerfile := range []string{
		"packages/vm-images/Dockerfile.base",
		"packages/vm-images/Dockerfile.uds-core",
	} {
		contents, err := os.ReadFile(dockerfile)
		if err != nil {
			t.Fatal(err)
		}
		text := string(contents)
		for _, required := range []string{"FROM python:", "USER 65532:65532", `CMD ["python3", "-m", "http.server"`} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s does not contain %q", dockerfile, required)
			}
		}
		if strings.Contains(text, "FROM scratch") {
			t.Fatalf("%s cannot serve HTTP from a scratch image", dockerfile)
		}
	}
}

func TestCDIExemptionIsOwnedByGoldenPVCComponent(t *testing.T) {
	serverCmd := exec.Command("helm", "template", "vm-images", "packages/vm-images/chart",
		"--set", "imageServers.enabled=true", "--set", "goldenPVCs.enabled=false")
	serverOut, err := serverCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render image-server chart: %v\n%s", err, serverOut)
	}
	if strings.Contains(string(serverOut), "name: uds-labs-vm-images-cdi") {
		t.Fatal("image-server component must not own the CDI exemption")
	}

	goldenCmd := exec.Command("helm", "template", "vm-images", "packages/vm-images/chart",
		"--set", "imageServers.enabled=false", "--set", "goldenPVCs.enabled=true")
	goldenOut, err := goldenCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render golden-pvcs chart: %v\n%s", err, goldenOut)
	}
	if !strings.Contains(string(goldenOut), "name: uds-labs-vm-images-cdi") {
		t.Fatal("golden-pvcs component must own the CDI exemption")
	}
}

func TestGoldenDataVolumesAreNotReconciledAfterImport(t *testing.T) {
	template := string(mustReadFile(t, "packages/vm-images/chart/templates/golden-pvcs.yaml"))
	for _, required := range []string{
		`lookup "cdi.kubevirt.io/v1beta1" "DataVolume" "uds-labs-vms"`,
		"if not $existing",
		"helm.sh/resource-policy: keep",
		"CDI rejects every DataVolume spec update",
	} {
		if !strings.Contains(template, required) {
			t.Fatalf("golden DataVolume template must contain %q", required)
		}
	}
}

func TestVMImageChartRendersPortablePlacementAndStorage(t *testing.T) {
	cmd := exec.Command("helm", "template", "vm-images", "packages/vm-images/chart",
		"--set-string", "imageServers.placement.nodeSelector.labs\\.uds\\.dev/compute=true",
		"--set", "imageServers.placement.tolerations[0].key=workload",
		"--set", "imageServers.placement.tolerations[0].operator=Equal",
		"--set", "imageServers.placement.tolerations[0].value=uds-labs",
		"--set", "imageServers.placement.tolerations[0].effect=NoSchedule",
		"--set", "goldenPVCs.storageClass=managed-csi-premium",
		"--set", "images.uds-core.size=64Gi")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("render VM image chart: %v\n%s", err, out)
	}
	text := string(out)
	for _, expected := range []string{
		"labs.uds.dev/compute: \"true\"",
		"value: uds-labs",
		"storageClassName: \"managed-csi-premium\"",
		"storage: 64Gi",
		"volumeMode: Filesystem",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("VM image chart missing %q", expected)
		}
	}
}

func TestCanonicalBundleOrdersDependenciesAndExposesPlacement(t *testing.T) {
	bundle := string(mustReadFile(t, "bundle/uds-bundle.yaml"))
	positions := []int{
		strings.Index(bundle, "name: kubevirt"),
		strings.Index(bundle, "name: cdi"),
		strings.Index(bundle, "name: uds-labs-vm-images"),
		strings.LastIndex(bundle, "name: uds-labs"),
	}
	for i, position := range positions {
		if position < 0 || (i > 0 && position <= positions[i-1]) {
			t.Fatal("canonical bundle must order KubeVirt, CDI, VM images, then UDS Labs")
		}
	}

	for _, requiredPackage := range []string{
		"name: kubevirt\n    repository: ghcr.io/uds-packages/kubevirt\n    ref: v1.9.0-uds.0-upstream",
		"name: cdi\n    repository: ghcr.io/uds-packages/cdi\n    ref: 1.65.0-uds.5-upstream",
	} {
		if !strings.Contains(bundle, requiredPackage) {
			t.Fatalf("development bundle is missing package reference %q", requiredPackage)
		}
	}
	for _, variable := range []string{
		"KUBEVIRT_OPERATOR_NODE_PLACEMENT", "KUBEVIRT_INFRA_NODE_PLACEMENT", "KUBEVIRT_WORKLOAD_NODE_PLACEMENT",
		"OPERATOR_NODE_PLACEMENT", "INFRA_NODE_PLACEMENT", "WORKLOAD_NODE_PLACEMENT",
		"LAB_SERVER_NODE_PLACEMENT", "LAB_OPERATOR_NODE_PLACEMENT", "LAB_VMI_NODE_PLACEMENT",
	} {
		if !strings.Contains(bundle, "name: "+variable) {
			t.Errorf("development bundle does not expose placement variable %q", variable)
		}
	}
	for _, forbidden := range []string{":latest", "node-role.kubernetes.io/control-plane", "node-role.kubernetes.io/master", "github.com/enxoco", "ghcr.io/enxoco"} {
		if strings.Contains(bundle, forbidden) {
			t.Errorf("development bundle contains forbidden value %q", forbidden)
		}
	}

	example := string(mustReadFile(t, "bundle/uds-config.example.yaml"))
	for _, variable := range []string{
		"KUBEVIRT_OPERATOR_NODE_PLACEMENT:", "KUBEVIRT_INFRA_NODE_PLACEMENT:", "KUBEVIRT_WORKLOAD_NODE_PLACEMENT:",
		"OPERATOR_NODE_PLACEMENT:", "INFRA_NODE_PLACEMENT:", "WORKLOAD_NODE_PLACEMENT:",
		"LAB_SERVER_NODE_PLACEMENT:", "LAB_OPERATOR_NODE_PLACEMENT:", "LAB_VMI_NODE_PLACEMENT:",
	} {
		if !strings.Contains(example, variable) {
			t.Errorf("portable deployment config example is missing %q", variable)
		}
	}
	if strings.Contains(example, "kubernetes.azure.com/agentpool") {
		t.Fatal("portable deployment config example must not prescribe an environment-specific node pool")
	}

	tasks := allTaskContents(t)
	for _, required := range []string{
		"Reusing staged VM-image package",
		`zarf package pull "oci://${REPOSITORY}:${VERSION}"`,
		"local: ${{ .inputs.LOCAL_VM_IMAGES }}",
		"UDS_CONFIG=\"$CONFIG\" uds deploy",
	} {
		if !strings.Contains(tasks, required) {
			t.Errorf("development tasks are missing %q", required)
		}
	}
}

func TestCanonicalBundleUsesPublishedInfrastructureAndVersionedApplication(t *testing.T) {
	bundle := string(mustReadFile(t, "bundle/uds-bundle.yaml"))
	var config struct {
		Metadata struct {
			Version string `yaml:"version"`
		} `yaml:"metadata"`
		Packages []struct {
			Name string `yaml:"name"`
			Ref  string `yaml:"ref"`
		} `yaml:"packages"`
	}
	readYAML(t, "bundle/uds-bundle.yaml", &config)
	for _, forbidden := range []string{"path: ../kubevirt", "path: ../cdi", "bundle/local", "kubernetes.azure.com/agentpool"} {
		if strings.Contains(bundle, forbidden) {
			t.Fatalf("canonical bundle contains forbidden local or environment-specific value %q", forbidden)
		}
	}
	for _, required := range []string{
		"repository: ghcr.io/uds-packages/kubevirt",
		"repository: ghcr.io/uds-packages/cdi",
		"path: ../packages/vm-images",
		"path: ../",
	} {
		if !strings.Contains(bundle, required) {
			t.Fatalf("canonical bundle is missing %q", required)
		}
	}
	for _, pkg := range config.Packages {
		if pkg.Name == "uds-labs" && pkg.Ref != config.Metadata.Version {
			t.Fatalf("bundle uds-labs ref %q does not match bundle version %q", pkg.Ref, config.Metadata.Version)
		}
	}

	tasks := allTaskContents(t)
	for _, forbidden := range []string{"BUNDLE_PROFILE", "bundle/local", "PROVIDER=pod", "PROVIDER: kubevirt"} {
		if strings.Contains(tasks, forbidden) {
			t.Errorf("tasks contain removed bundle/provider behavior %q", forbidden)
		}
	}
	for _, required := range []string{
		"name: deploy-local-uds-core",
		"bundle/core-base-values.yaml",
		"bundle/core-identity-values.yaml",
		"name: local-e2e",
		"task: access:test-session-lifecycle",
	} {
		if !strings.Contains(tasks, required) {
			t.Errorf("local E2E tasks are missing %q", required)
		}
	}

	baseValues := string(mustReadFile(t, "bundle/core-base-values.yaml"))
	identityValues := string(mustReadFile(t, "bundle/core-identity-values.yaml"))
	for _, required := range []string{"istio-controlplane:", "pepr-uds-core:"} {
		if !strings.Contains(baseValues, required) {
			t.Errorf("local UDS Core base values are missing %q", required)
		}
	}
	for _, required := range []string{"authservice:", "replicaCount: 1", "insecureAdminPasswordGeneration:"} {
		if !strings.Contains(identityValues, required) {
			t.Errorf("local UDS Core identity values are missing %q", required)
		}
	}
}

func TestRootTaskContractMatchesUDSPackageConventions(t *testing.T) {
	var config taskConfig
	readYAML(t, "tasks.yaml", &config)
	definitions := map[string]*taskDefinition{}
	for i := range config.Tasks {
		definitions[config.Tasks[i].Name] = &config.Tasks[i]
	}
	for _, required := range []string{
		"default",
		"create-dev-package",
		"create-deploy-test-bundle",
		"dev",
		"test-install",
		"test-upgrade",
		"publish-package",
	} {
		if definitions[required] == nil {
			t.Fatalf("root task contract is missing %q", required)
		}
	}
	if definitions["dev"].Actions[0].Task != "test:dev" {
		t.Fatalf("root dev task = %q, want test:dev", definitions["dev"].Actions[0].Task)
	}
}

func TestLintCompatibilityTaskRemainsDirectlyRunnable(t *testing.T) {
	lint := taskDefinitions(t)["lint"]
	if lint == nil || len(lint.Actions) != 1 || lint.Actions[0].Cmd != "golangci-lint run ./..." {
		t.Fatal("root lint compatibility task must run golangci-lint directly for attest:lint")
	}
}

func TestUDSTasksDoNotShellOutToRepositoryScripts(t *testing.T) {
	if _, err := os.Stat("scripts"); !os.IsNotExist(err) {
		t.Fatalf("root scripts directory must be removed; use tasks/*.yaml instead (err: %v)", err)
	}
	if strings.Contains(allTaskContents(t), "scripts/") {
		t.Fatal("UDS tasks must not invoke shell scripts from the repository scripts directory")
	}
}

func TestPackerDownloadsRetryConnectionFailures(t *testing.T) {
	contents, err := os.ReadFile("packer/scripts/base.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(contents)
	for _, required := range []string{
		"curl_retry()",
		"--retry-all-errors",
		"--connect-timeout",
		"--max-time",
	} {
		if !strings.Contains(script, required) {
			t.Fatalf("packer download helper does not contain %q", required)
		}
	}
	if count := strings.Count(script, "curl -fsSL"); count != 1 {
		t.Fatalf("all packer downloads must use curl_retry; found %d direct curl invocations", count)
	}
}

func TestApplicationImageAndVersionsStayConsistent(t *testing.T) {
	var pkg applicationPackage
	readYAML(t, "zarf.yaml", &pkg)

	var values chartValues
	readYAML(t, "chart/values.yaml", &values)

	var chart chartMetadata
	readYAML(t, "chart/Chart.yaml", &chart)

	for _, variable := range pkg.Variables {
		if variable.Name == "IMAGE" {
			t.Fatal("image must not be a deployment variable because an override can name an image absent from the package")
		}
	}

	for _, component := range pkg.Components {
		if component.Name != "uds-labs" {
			continue
		}
		if len(component.Images) != 2 {
			t.Fatalf("uds-labs component must package the application and capacity-bootstrap images, got %d", len(component.Images))
		}
		if component.Images[0] != values.Image {
			t.Fatalf("Zarf application image %q differs from chart image %q", component.Images[0], values.Image)
		}
		if component.Images[1] != "registry.k8s.io/pause:3.10" {
			t.Fatalf("Zarf capacity-bootstrap image = %q, want registry.k8s.io/pause:3.10", component.Images[1])
		}
		if !strings.HasPrefix(values.Image, "ghcr.io/defenseunicorns-labs/") {
			t.Fatalf("application image must use defenseunicorns-labs GHCR, got %q", values.Image)
		}
		if strings.HasSuffix(values.Image, ":latest") {
			t.Fatal("application image must use an immutable version tag")
		}
		if len(component.Charts) != 1 || component.Charts[0].Version != chart.Version {
			t.Fatalf("Zarf chart version %q must match chart/Chart.yaml version %q", component.Charts[0].Version, chart.Version)
		}
		if chart.AppVersion == "" {
			t.Fatal("chart appVersion must be set")
		}
		if pkg.Metadata.Version != "dev" {
			wantImage := "ghcr.io/defenseunicorns-labs/uds-labs:" + pkg.Metadata.Version
			if values.Image != wantImage {
				t.Fatalf("release image %q, want %q", values.Image, wantImage)
			}
			if chart.Version != pkg.Metadata.Version || chart.AppVersion != pkg.Metadata.Version {
				t.Fatalf("release chart versions = version:%q appVersion:%q, want %q", chart.Version, chart.AppVersion, pkg.Metadata.Version)
			}
		}
		return
	}

	t.Fatal("application package has no uds-labs component")
}

func TestReleasePreparationSynchronizesRepositoryOwnedArtifacts(t *testing.T) {
	contents := string(mustReadFile(t, "tasks/release.yaml"))
	for _, required := range []string{
		"uds-pk release update-yaml",
		"chart/Chart.yaml",
		"chart/values.yaml",
		"bundle/uds-bundle.yaml",
		"ghcr.io/defenseunicorns-labs/uds-labs:${VERSION}",
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("release preparation is missing %q", required)
		}
	}
}

func TestReleasePublisherUsesTheBuiltArchiveVersionAndFlavor(t *testing.T) {
	contents := string(mustReadFile(t, "tasks.yaml"))
	for _, required := range []string{
		"name: PACKAGE_VERSION",
		"task: publish:release-please-publish",
		"version: ${{ .variables.PACKAGE_VERSION }}",
		"task: publish:uds-pk-publish",
		"flavor: ${{ .inputs.flavor }}",
	} {
		if !strings.Contains(contents, required) {
			t.Fatalf("release publisher is missing %q", required)
		}
	}
}

func TestReleasePromotesOneValidatedArchive(t *testing.T) {
	contents := string(mustReadFile(t, "tasks.yaml"))
	start := strings.Index(contents, "- name: release")
	if start < 0 {
		t.Fatal("release flow is missing the release task")
	}
	contents = contents[start:]
	markers := []string{
		"- name: release",
		"Attest repository lint",
		"Run security scans",
		"Build the application image used by the Zarf package",
		"Build the signed Zarf package",
		"Record the exact Zarf archive produced for this release",
		"Validate the built package on k3d before publication",
		"Publish the validated release archive to GHCR and create the GitHub release",
		"Vouch the package and attestations to CAT",
		"Publish the vouched release archive to the UDS Proving Ground registry",
	}
	previous := -1
	for _, marker := range markers {
		index := strings.Index(contents, marker)
		if index < 0 {
			t.Fatalf("release flow is missing %q", marker)
		}
		if index <= previous {
			t.Fatalf("release step %q is out of order", marker)
		}
		previous = index
	}
	if !strings.Contains(contents, "zarf_package: ${{ .variables.PACKAGE_ARCHIVE }}") {
		t.Fatal("release must publish the explicitly recorded Zarf archive")
	}
	if _, err := os.Stat(".github/workflows/release.yaml"); err != nil {
		t.Fatalf("primary release workflow is missing: %v", err)
	}
	if _, err := os.Stat(".github/workflows/udm-release.yaml"); !os.IsNotExist(err) {
		t.Fatalf("legacy UDM release workflow must be removed (err: %v)", err)
	}
}

func readYAML(t *testing.T, path string, target any) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(contents, target); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
}
