// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package scenario

import (
	"os"
	"testing"
	"testing/fstest"
)

func TestLoadCatalogResolvesScenariosAndResourcesInConfiguredOrder(t *testing.T) {
	fsys := fstest.MapFS{
		"catalog.yaml": {Data: []byte(`sections:
  - id: learn
    title: Learn
    description: Build a foundation.
    items:
      - scenario: app
      - resource:
          id: docs
          title: Official docs
          description: Read the guide.
          url: https://example.com/docs
          outcome: Understand the platform.
          prerequisites:
            - None
          duration: 10
          difficulty: beginner
`)},
		"app/scenario.yaml": {Data: []byte(`title: Package an app
description: Build and deploy it.
outcome: Deploy a working package.
prerequisites:
  - Containerized application
  - Basic Helm
duration: 45
difficulty: intermediate
steps:
  - title: Start
    text: steps/start.md
`)},
		"app/steps/start.md": {Data: []byte("# Start")},
	}

	catalog, err := LoadCatalog(fsys)
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	if len(catalog.Sections) != 1 || len(catalog.Sections[0].Items) != 2 {
		t.Fatalf("unexpected catalog shape: %#v", catalog)
	}

	scenarioItem := catalog.Sections[0].Items[0]
	if scenarioItem.Type != CatalogItemScenario || scenarioItem.ID != "app" {
		t.Fatalf("scenario item = %#v", scenarioItem)
	}
	if scenarioItem.Outcome != "Deploy a working package." || len(scenarioItem.Prerequisites) != 2 {
		t.Fatalf("scenario learning metadata not resolved: %#v", scenarioItem)
	}

	resourceItem := catalog.Sections[0].Items[1]
	if resourceItem.Type != CatalogItemResource || resourceItem.ID != "docs" {
		t.Fatalf("resource item = %#v", resourceItem)
	}
	if resourceItem.URL != "https://example.com/docs" || resourceItem.Duration != 10 {
		t.Fatalf("resource metadata not preserved: %#v", resourceItem)
	}
}

func TestRepositoryCatalogLoads(t *testing.T) {
	catalog, err := LoadCatalog(os.DirFS("../../scenarios"))
	if err != nil {
		t.Fatalf("LoadCatalog(repository) error = %v", err)
	}
	if len(catalog.Sections) != 2 {
		t.Fatalf("catalog sections = %d, want 2", len(catalog.Sections))
	}
}

func TestLoadCatalogRejectsUnknownScenario(t *testing.T) {
	fsys := fstest.MapFS{
		"catalog.yaml": {Data: []byte(`sections:
  - id: learn
    title: Learn
    items:
      - scenario: missing
`)},
	}

	if _, err := LoadCatalog(fsys); err == nil {
		t.Fatal("LoadCatalog() expected an error for an unknown scenario")
	}
}

func TestLoadCatalogRejectsAmbiguousItem(t *testing.T) {
	fsys := fstest.MapFS{
		"catalog.yaml": {Data: []byte(`sections:
  - id: learn
    title: Learn
    items:
      - scenario: app
        resource:
          id: docs
          title: Docs
          url: https://example.com
`)},
	}

	if _, err := LoadCatalog(fsys); err == nil {
		t.Fatal("LoadCatalog() expected an error when an item has both scenario and resource")
	}
}
