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

func TestVMImageComponentsInDedicatedPackage(t *testing.T) {
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

func TestDevelopmentBundleOrdersDependenciesAndExposesPlacement(t *testing.T) {
	bundle := string(mustReadFile(t, "bundle/uds-bundle.yaml"))
	positions := []int{
		strings.Index(bundle, "name: kubevirt"),
		strings.Index(bundle, "name: cdi"),
		strings.Index(bundle, "name: uds-labs-vm-images"),
		strings.LastIndex(bundle, "name: uds-labs"),
	}
	for i, position := range positions {
		if position < 0 || (i > 0 && position <= positions[i-1]) {
			t.Fatal("development bundle must order KubeVirt, CDI, VM images, then UDS Labs")
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

func TestLocalE2EProfileUsesK3sDefaults(t *testing.T) {
	profile := string(mustReadFile(t, "bundle/local/uds-bundle.yaml"))
	for _, required := range []string{
		"name: uds-labs-local",
		"path: goldenPVCs.storageClass",
		"value: local-path",
		"path: storageClass",
		"path: auth.allowedGroups",
		"value: []",
	} {
		if !strings.Contains(profile, required) {
			t.Errorf("local bundle profile is missing %q", required)
		}
	}
	for _, forbidden := range []string{
		"kubernetes.azure.com/agentpool",
		"managed-csi-premium",
		"/UDS Labs/Pilot",
	} {
		if strings.Contains(profile, forbidden) {
			t.Errorf("local bundle profile contains environment-specific value %q", forbidden)
		}
	}

	tasks := allTaskContents(t)
	for _, required := range []string{
		"name: deploy-local-uds-core",
		"ghcr.io/defenseunicorns/packages/uds/core-base:1.10.0-upstream",
		"ghcr.io/defenseunicorns/packages/uds/core-identity-authorization:1.10.0-upstream",
		"uds zarf package deploy",
		"bundle/local/core-base-values.yaml",
		"bundle/local/core-identity-values.yaml",
		"name: local-e2e",
		"BUNDLE_PROFILE: local",
		"task: access:patch-coredns",
		"task: access:start-proxy",
		"task: access:create-test-user",
	} {
		if !strings.Contains(tasks, required) {
			t.Errorf("local E2E tasks are missing %q", required)
		}
	}
	if strings.Contains(tasks, "k3d-core-slim-dev") {
		t.Error("bare-k3s local E2E must not reference the k3d UDS Core bundle")
	}

	baseValues := string(mustReadFile(t, "bundle/local/core-base-values.yaml"))
	identityValues := string(mustReadFile(t, "bundle/local/core-identity-values.yaml"))
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

func TestDeployStackPatchesAndChecksDemoRoutesAfterPackageReconciliation(t *testing.T) {
	definitions := taskDefinitions(t)
	for _, required := range []string{"local-e2e", "deploy-stack", "patch-demo-routes", "smoke-test"} {
		if definitions[required] == nil {
			t.Fatalf("task files must define %q", required)
		}
	}

	actionPositions := func(task *taskDefinition) map[string]int {
		positions := map[string]int{}
		for i, action := range task.Actions {
			if action.Task != "" {
				name := action.Task
				if separator := strings.LastIndex(name, ":"); separator >= 0 {
					name = name[separator+1:]
				}
				positions[name] = i
			}
		}
		return positions
	}
	localPositions := actionPositions(definitions["local-e2e"])
	for _, required := range []string{"deploy-stack", "patch-coredns"} {
		if _, exists := localPositions[required]; !exists {
			t.Fatalf("local-e2e is missing task %q", required)
		}
	}
	if localPositions["deploy-stack"] >= localPositions["patch-coredns"] {
		t.Fatalf("local-e2e must deploy the stack before configuring local access; positions: %#v", localPositions)
	}

	deployPositions := actionPositions(definitions["deploy-stack"])
	for _, required := range []string{"deploy-bundle", "patch-demo-routes", "wait-kubevirt"} {
		if _, exists := deployPositions[required]; !exists {
			t.Fatalf("deploy-stack is missing task %q", required)
		}
	}
	if deployPositions["deploy-bundle"] >= deployPositions["patch-demo-routes"] ||
		deployPositions["patch-demo-routes"] >= deployPositions["wait-kubevirt"] {
		t.Fatalf("deploy-stack must patch demo routes immediately after bundle deployment; positions: %#v", deployPositions)
	}

	var patchCommands strings.Builder
	for _, action := range definitions["patch-demo-routes"].Actions {
		patchCommands.WriteString(action.Cmd)
	}
	if !strings.Contains(patchCommands.String(), ".status.observedGeneration") {
		t.Fatal("patch-demo-routes must wait for the latest Package reconciliation before patching Pepr-managed policies")
	}

	var smokeCommands strings.Builder
	for _, action := range definitions["smoke-test"].Actions {
		smokeCommands.WriteString(action.Cmd)
	}
	for _, required := range []string{"uds-labs-authservice", "uds-labs-jwt-authz", "/api/demo-sessions"} {
		if !strings.Contains(smokeCommands.String(), required) {
			t.Fatalf("smoke-test does not verify demo-route policy fragment %q", required)
		}
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
		if len(component.Images) != 1 {
			t.Fatalf("uds-labs component must package exactly one application image, got %d", len(component.Images))
		}
		if component.Images[0] != values.Image {
			t.Fatalf("Zarf image %q differs from chart image %q", component.Images[0], values.Image)
		}
		if !strings.HasPrefix(values.Image, "ghcr.io/defenseunicorns-labs/") {
			t.Fatalf("application image must use defenseunicorns-labs GHCR, got %q", values.Image)
		}
		if strings.HasSuffix(values.Image, ":latest") {
			t.Fatal("application image must use an immutable version tag")
		}
		if !strings.HasSuffix(values.Image, ":"+pkg.Metadata.Version) {
			t.Fatalf("application image %q must be tagged with package version %q", values.Image, pkg.Metadata.Version)
		}
		if chart.Version != pkg.Metadata.Version || chart.AppVersion != pkg.Metadata.Version {
			t.Fatalf("package version %q, chart version %q, and appVersion %q must match", pkg.Metadata.Version, chart.Version, chart.AppVersion)
		}
		if len(component.Charts) != 1 || component.Charts[0].Version != chart.Version {
			t.Fatal("Zarf chart version must match chart/Chart.yaml")
		}
		return
	}

	t.Fatal("application package has no uds-labs component")
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
