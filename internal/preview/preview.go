// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

// Package preview provides the small, cluster-free scenario authoring runtime.
package preview

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/defenseunicorns-labs/uds-labs/internal/scenario"
)

// Result reports the outcome of one preview verification request.
type Result struct {
	Passed   bool     `json:"passed"`
	Messages []string `json:"messages"`
}

// Validate checks a scenario's optional preview contract. Scenarios without a
// preview remain valid live-cluster scenarios.
func Validate(fsys fs.FS, s *scenario.Scenario) error {
	if s.Preview == nil {
		return nil
	}
	p := s.Preview
	if p.Fixture == "" || p.Workspace == "" {
		return errors.New("preview requires fixture and workspace")
	}
	if path.IsAbs(p.Fixture) || !validRelativePath(p.Fixture) {
		return fmt.Errorf("preview fixture %q must be a relative path", p.Fixture)
	}
	if _, err := fs.Stat(fsys, path.Join(s.ID, p.Fixture)); err != nil {
		return fmt.Errorf("preview fixture %q: %w", p.Fixture, err)
	}
	if len(p.Checks) == 0 {
		return errors.New("preview requires at least one check")
	}
	for i, check := range p.Checks {
		if check.Step < 1 || check.Step > len(s.Steps) {
			return fmt.Errorf("preview check %d has invalid step %d", i+1, check.Step)
		}
		if !validRelativePath(check.Path) {
			return fmt.Errorf("preview check %d path %q must be relative", i+1, check.Path)
		}
		if len(check.Contains) == 0 {
			return fmt.Errorf("preview check %d requires at least one contains assertion", i+1)
		}
	}
	return nil
}

// Prepare replaces workspace with a writable copy of the scenario fixture.
func Prepare(fsys fs.FS, s *scenario.Scenario, workspace string) error {
	if err := Validate(fsys, s); err != nil {
		return err
	}
	if workspace == "" {
		workspace = s.Preview.Workspace
	}
	if !filepath.IsAbs(workspace) {
		return fmt.Errorf("workspace %q must be absolute", workspace)
	}
	if err := os.RemoveAll(workspace); err != nil {
		return fmt.Errorf("remove workspace: %w", err)
	}
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	source := path.Join(s.ID, s.Preview.Fixture)
	return fs.WalkDir(fsys, source, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(sourcePath, source)
		rel = strings.TrimPrefix(rel, "/")
		destination := filepath.Join(workspace, filepath.FromSlash(rel))
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		contents, err := fs.ReadFile(fsys, sourcePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		return os.WriteFile(destination, contents, 0o644)
	})
}

// Verify runs the checks for a single step. A step with no checks is treated as
// informational and passes; this lets scenarios include reading-only steps.
func Verify(s *scenario.Scenario, workspace string, step int) Result {
	if s.Preview == nil {
		return Result{Messages: []string{"This scenario does not provide cluster-free preview checks."}}
	}
	if workspace == "" {
		workspace = s.Preview.Workspace
	}
	result := Result{Passed: true}
	for _, check := range s.Preview.Checks {
		if check.Step != step {
			continue
		}
		contents, err := os.ReadFile(filepath.Join(workspace, filepath.FromSlash(check.Path)))
		if err != nil {
			result.Passed = false
			result.Messages = append(result.Messages, fmt.Sprintf("%s is missing", check.Path))
			continue
		}
		for _, expected := range check.Contains {
			if !strings.Contains(string(contents), expected) {
				result.Passed = false
				result.Messages = append(result.Messages, fmt.Sprintf("%s must contain %q", check.Path, expected))
			}
		}
	}
	if result.Passed {
		result.Messages = append(result.Messages, "Preview checks passed.")
	}
	return result
}

// VerifyAll requires every configured check to pass.
func VerifyAll(s *scenario.Scenario, workspace string) Result {
	if s.Preview == nil {
		return Result{Messages: []string{"This scenario does not provide cluster-free preview checks."}}
	}
	result := Result{Passed: true}
	for step := range s.Steps {
		stepResult := Verify(s, workspace, step+1)
		if !stepResult.Passed {
			result.Passed = false
			result.Messages = append(result.Messages, stepResult.Messages...)
		}
	}
	if result.Passed {
		result.Messages = []string{"All preview checks passed."}
	}
	return result
}

func validRelativePath(value string) bool {
	clean := path.Clean(value)
	return value != "" && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}
