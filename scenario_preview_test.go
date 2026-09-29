// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package labplatform_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/defenseunicorns-labs/uds-labs/internal/preview"
	"github.com/defenseunicorns-labs/uds-labs/internal/scenario"
)

func TestScenarioPreviewContracts(t *testing.T) {
	scenariosFS := os.DirFS("scenarios")
	entries, err := os.ReadDir("scenarios")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		s, err := scenario.Load(scenariosFS, entry.Name())
		if err != nil {
			t.Errorf("%s: load: %v", entry.Name(), err)
			continue
		}
		if err := preview.Validate(scenariosFS, s); err != nil {
			t.Errorf("%s: preview contract: %v", entry.Name(), err)
		}
	}
}

func TestReleaseLedgerPostgresPreviewChecks(t *testing.T) {
	scenariosFS := os.DirFS("scenarios")
	s, err := scenario.Load(scenariosFS, "release-ledger-postgres")
	if err != nil {
		t.Fatal(err)
	}

	workspace := filepath.Join(t.TempDir(), "release-ledger-postgres")
	if err := preview.Prepare(scenariosFS, s, workspace); err != nil {
		t.Fatal(err)
	}
	if result := preview.VerifyAll(s, workspace); !result.Passed {
		t.Fatalf("preview checks failed: %#v", result)
	}
}

func TestReleaseLedgerPostgresBundleUsesOperatorNamespaces(t *testing.T) {
	contents, err := os.ReadFile("scenarios/release-ledger-postgres/app/bundle/uds-bundle.yaml")
	if err != nil {
		t.Fatal(err)
	}
	bundle := string(contents)
	if strings.Contains(bundle, "namespace: release-ledger-postgres-db") {
		t.Fatal("postgres-operator is multi-namespace and must not receive a bundle namespace override")
	}
	if !strings.Contains(bundle, "remoteNamespace: release-ledger-postgres") {
		t.Fatal("postgres ingress must allow the application namespace")
	}

	values, err := os.ReadFile("scenarios/release-ledger-postgres/app/chart/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(values), "databaseHost: pg-cluster.postgres.svc.cluster.local") {
		t.Fatal("application must use the operator package's postgres namespace")
	}
}

func TestReleaseLedgerPostgresStep3Checkpoint(t *testing.T) {
	workspace, err := filepath.Abs("scenarios/release-ledger-postgres/app")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "scenarios/release-ledger-postgres/verify/step3.sh")
	cmd.Env = append(os.Environ(), "SCENARIO_WORKSPACE="+workspace)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("step 3 checkpoint failed: %v\n%s", err, output)
	}
}

func TestReleaseLedgerMinioInstructionsUseDefaultImageToolsAndWorkspaceRoot(t *testing.T) {
	for _, path := range []string{
		"scenarios/release-ledger-minio/intro.md",
		"scenarios/release-ledger-minio/steps/1-storage-contract.md",
	} {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(contents)
		if strings.Contains(text, "app/app.py") {
			t.Fatalf("%s must reference the workspace-root app.py, not app/app.py", path)
		}
		if strings.Contains(text, "rg ") {
			t.Fatalf("%s must use tools available in the default image, not rg", path)
		}
	}
}

func TestReleaseLedgerMinioTenantUsesMatchingConfigSecret(t *testing.T) {
	contents, err := os.ReadFile("scenarios/release-ledger-minio/app/bundle/uds-bundle.yaml")
	if err != nil {
		t.Fatal(err)
	}
	bundle := string(contents)
	if !strings.Contains(bundle, "secretName: release-ledger-minio") {
		t.Fatal("MinIO app credentials must be copied to the application Secret")
	}
	if !strings.Contains(bundle, "secretIDKey: access_key") || !strings.Contains(bundle, "secretPasswordKey: secret_key") {
		t.Fatal("MinIO app Secret keys must match the application configuration")
	}
}

func TestReleaseLedgerMinioUsesOperatorDefaultTenant(t *testing.T) {
	bundleContents, err := os.ReadFile("scenarios/release-ledger-minio/app/bundle/uds-bundle.yaml")
	if err != nil {
		t.Fatal(err)
	}
	bundle := string(bundleContents)
	for _, override := range []string{
		"path: tenant.name",
		"path: tenant.configuration.name",
		"path: tenant.configSecret.name",
	} {
		if strings.Contains(bundle, override) {
			t.Fatalf("bundle must not rename the MinIO operator's built-in tenant: found %q", override)
		}
	}

	valuesContents, err := os.ReadFile("scenarios/release-ledger-minio/app/chart/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(valuesContents), "s3Endpoint: uds-minio-hl.minio.svc.cluster.local:9000") {
		t.Fatal("application must use the MinIO operator's built-in tenant service")
	}

	packageContents, err := os.ReadFile("scenarios/release-ledger-minio/app/chart/templates/uds-package.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(packageContents), "v1.min.io/tenant: uds-minio") {
		t.Fatal("application network policy must target the built-in MinIO tenant")
	}
}

func TestReleaseLedgerMinioUsesPostgresOperatorNamespace(t *testing.T) {
	valuesContents, err := os.ReadFile("scenarios/release-ledger-minio/app/chart/values.yaml")
	if err != nil {
		t.Fatal(err)
	}
	values := string(valuesContents)
	if !strings.Contains(values, "databaseHost: pg-cluster.postgres.svc.cluster.local") {
		t.Fatal("application must use the postgres operator's fixed postgres namespace")
	}
	if !strings.Contains(values, "databaseSecretName: release-ledger-minio.release-ledger.pg-cluster.credentials.postgresql.acid.zalan.do") {
		t.Fatal("application must use the generated Secret in the postgres namespace")
	}

	packageContents, err := os.ReadFile("scenarios/release-ledger-minio/app/chart/templates/uds-package.yaml")
	if err != nil {
		t.Fatal(err)
	}
	packageText := string(packageContents)
	if !strings.Contains(packageText, "remoteNamespace: postgres") {
		t.Fatal("application network policy must target the postgres operator namespace")
	}
	if strings.Contains(packageText, "remoteNamespace: release-ledger-minio-db") {
		t.Fatal("application must not target the nonexistent release-ledger-minio-db namespace")
	}
}

func TestReleaseLedgerMinioUsesValidStandalonePool(t *testing.T) {
	contents, err := os.ReadFile("scenarios/release-ledger-minio/app/bundle/uds-bundle.yaml")
	if err != nil {
		t.Fatal(err)
	}
	bundle := string(contents)
	if !strings.Contains(bundle, "servers: 1") || !strings.Contains(bundle, "volumesPerServer: 4") {
		t.Fatal("MinIO standalone pool must use one server with four volumes")
	}
	if strings.Contains(bundle, "volumesPerServer: 1") {
		t.Fatal("MinIO standalone pool must not use one volume")
	}
}

func TestReleaseLedgerMinioStep1Checkpoint(t *testing.T) {
	workspace, err := filepath.Abs("scenarios/release-ledger-minio/app")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "scenarios/release-ledger-minio/verify/step1.sh")
	cmd.Env = append(os.Environ(), "SCENARIO_WORKSPACE="+workspace)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("step 1 checkpoint failed: %v\n%s", err, output)
	}
}
