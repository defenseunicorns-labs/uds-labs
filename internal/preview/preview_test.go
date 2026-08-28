// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package preview

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/defenseunicorns-labs/uds-labs/internal/scenario"
)

func TestPrepareAndVerify(t *testing.T) {
	fsys := fstest.MapFS{
		"app/preview/fixture/config.yaml": {Data: []byte("enabled: true\n")},
	}
	s := &scenario.Scenario{
		ID:    "app",
		Steps: []scenario.Step{{Title: "Configure"}},
		Preview: &scenario.Preview{
			Fixture:   "preview/fixture",
			Workspace: "/workspace/app",
			Checks: []scenario.PreviewCheck{{
				Step:     1,
				Path:     "config.yaml",
				Contains: []string{"enabled: true"},
			}},
		},
	}
	workspace := filepath.Join(t.TempDir(), "app")
	if err := Prepare(fsys, s, workspace); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(workspace, "config.yaml")); err != nil {
		t.Fatalf("fixture was not copied: %v", err)
	}
	if result := Verify(s, workspace, 1); !result.Passed {
		t.Fatalf("Verify() = %#v, want pass", result)
	}
}

func TestVerifyReportsMissingExpectedContent(t *testing.T) {
	s := &scenario.Scenario{
		Preview: &scenario.Preview{Checks: []scenario.PreviewCheck{{
			Step: 1, Path: "config.yaml", Contains: []string{"enabled: true"},
		}}},
	}
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, "config.yaml"), []byte("enabled: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := Verify(s, workspace, 1)
	if result.Passed || len(result.Messages) != 1 {
		t.Fatalf("Verify() = %#v, want one failed assertion", result)
	}
}

func TestValidateRejectsEscapingFixture(t *testing.T) {
	s := &scenario.Scenario{
		ID:    "app",
		Steps: []scenario.Step{{Title: "Configure"}},
		Preview: &scenario.Preview{Fixture: "../fixture", Workspace: "/workspace/app", Checks: []scenario.PreviewCheck{{
			Step: 1, Path: "config.yaml", Contains: []string{"enabled"},
		}}},
	}
	if err := Validate(fstest.MapFS{}, s); err == nil {
		t.Fatal("Validate() unexpectedly accepted an escaping fixture")
	}
}
