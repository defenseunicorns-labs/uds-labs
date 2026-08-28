// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

package scenario

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

const (
	CatalogItemScenario = "scenario"
	CatalogItemResource = "resource"
)

type Catalog struct {
	Sections []CatalogSection `json:"sections"`
}

type CatalogSection struct {
	ID          string        `yaml:"id"          json:"id"`
	Title       string        `yaml:"title"       json:"title"`
	Description string        `yaml:"description" json:"description"`
	Items       []CatalogItem `yaml:"-"           json:"items"`
}

type CatalogItem struct {
	Type          string   `json:"type"`
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Outcome       string   `json:"outcome"`
	Prerequisites []string `json:"prerequisites"`
	Duration      int      `json:"duration"`
	Difficulty    string   `json:"difficulty"`
	Playground    bool     `json:"playground"`
	URL           string   `json:"url,omitempty"`
}

type catalogFile struct {
	Sections []catalogSectionFile `yaml:"sections"`
}

type catalogSectionFile struct {
	ID          string            `yaml:"id"`
	Title       string            `yaml:"title"`
	Description string            `yaml:"description"`
	Items       []catalogItemFile `yaml:"items"`
}

type catalogItemFile struct {
	Scenario string           `yaml:"scenario"`
	Resource *CatalogResource `yaml:"resource"`
}

type CatalogResource struct {
	ID            string   `yaml:"id"`
	Title         string   `yaml:"title"`
	Description   string   `yaml:"description"`
	URL           string   `yaml:"url"`
	Outcome       string   `yaml:"outcome"`
	Prerequisites []string `yaml:"prerequisites"`
	Duration      int      `yaml:"duration"`
	Difficulty    string   `yaml:"difficulty"`
}

func LoadCatalog(fsys fs.FS) (*Catalog, error) {
	data, err := fs.ReadFile(fsys, "catalog.yaml")
	if err != nil {
		return nil, fmt.Errorf("read catalog.yaml: %w", err)
	}

	var config catalogFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse catalog.yaml: %w", err)
	}
	if len(config.Sections) == 0 {
		return nil, fmt.Errorf("catalog.yaml has no sections")
	}

	catalog := &Catalog{Sections: make([]CatalogSection, 0, len(config.Sections))}
	sectionIDs := map[string]bool{}
	itemIDs := map[string]bool{}
	for sectionIndex, configuredSection := range config.Sections {
		if configuredSection.ID == "" || configuredSection.Title == "" {
			return nil, fmt.Errorf("catalog section %d requires id and title", sectionIndex+1)
		}
		if sectionIDs[configuredSection.ID] {
			return nil, fmt.Errorf("duplicate catalog section id %q", configuredSection.ID)
		}
		sectionIDs[configuredSection.ID] = true

		section := CatalogSection{
			ID:          configuredSection.ID,
			Title:       configuredSection.Title,
			Description: configuredSection.Description,
			Items:       make([]CatalogItem, 0, len(configuredSection.Items)),
		}
		for itemIndex, configuredItem := range configuredSection.Items {
			hasScenario := configuredItem.Scenario != ""
			hasResource := configuredItem.Resource != nil
			if hasScenario == hasResource {
				return nil, fmt.Errorf("catalog section %q item %d must define exactly one of scenario or resource", configuredSection.ID, itemIndex+1)
			}

			var item CatalogItem
			if hasScenario {
				s, err := Load(fsys, configuredItem.Scenario)
				if err != nil {
					return nil, fmt.Errorf("catalog section %q: %w", configuredSection.ID, err)
				}
				item = CatalogItem{
					Type:          CatalogItemScenario,
					ID:            s.ID,
					Title:         s.Title,
					Description:   s.Description,
					Outcome:       s.Outcome,
					Prerequisites: s.Prerequisites,
					Duration:      s.Duration,
					Difficulty:    s.Difficulty,
					Playground:    s.Playground,
				}
			} else {
				resource := configuredItem.Resource
				if resource.ID == "" || resource.Title == "" || resource.URL == "" {
					return nil, fmt.Errorf("catalog section %q resource %d requires id, title, and url", configuredSection.ID, itemIndex+1)
				}
				item = CatalogItem{
					Type:          CatalogItemResource,
					ID:            resource.ID,
					Title:         resource.Title,
					Description:   resource.Description,
					Outcome:       resource.Outcome,
					Prerequisites: resource.Prerequisites,
					Duration:      resource.Duration,
					Difficulty:    resource.Difficulty,
					URL:           resource.URL,
				}
			}
			if item.Description == "" || item.Outcome == "" || len(item.Prerequisites) == 0 || item.Duration <= 0 || item.Difficulty == "" {
				return nil, fmt.Errorf("catalog item %q requires description, outcome, prerequisites, positive duration, and difficulty", item.ID)
			}
			if itemIDs[item.ID] {
				return nil, fmt.Errorf("duplicate catalog item id %q", item.ID)
			}
			itemIDs[item.ID] = true
			section.Items = append(section.Items, item)
		}
		catalog.Sections = append(catalog.Sections, section)
	}

	return catalog, nil
}
