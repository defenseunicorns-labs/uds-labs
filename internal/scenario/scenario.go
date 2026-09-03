package scenario

import (
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

type ServiceLink struct {
	Label string `yaml:"label" json:"label"`
	URL   string `yaml:"url"   json:"url"`
}

type Step struct {
	Title     string `yaml:"title"  json:"title"`
	Text      string `yaml:"text"   json:"-"`
	Verify    string `yaml:"verify" json:"-"`
	Content   string `yaml:"-"      json:"content"`
	HasVerify bool   `yaml:"-"      json:"has_verify"`
}

// Preview describes a cluster-free authoring environment for a scenario. The
// fixture is copied into Workspace before the learner runs the documented steps.
type Preview struct {
	Fixture   string         `yaml:"fixture"   json:"-"`
	Workspace string         `yaml:"workspace" json:"-"`
	Checks    []PreviewCheck `yaml:"checks"    json:"-"`
}

// PreviewCheck is a small, declarative assertion over a learner-owned file.
// It intentionally validates artifacts rather than trying to emulate UDS or
// Kubernetes in full.
type PreviewCheck struct {
	Step     int      `yaml:"step"     json:"-"`
	Path     string   `yaml:"path"     json:"-"`
	Contains []string `yaml:"contains" json:"-"`
}

// Orientation is the structured first-time briefing shown before a lab.
// The UI owns the page sequence; scenario authors provide the content.
type Orientation struct {
	Mission        string             `yaml:"mission"         json:"mission"`
	Why            string             `yaml:"why"             json:"why,omitempty"`
	StartingPoint  StartingPoint      `yaml:"starting_point"  json:"starting_point"`
	Journey        []OrientationStep  `yaml:"journey"         json:"journey"`
	Success        OrientationSuccess `yaml:"success"         json:"success"`
	Tools          []string           `yaml:"tools"           json:"tools"`
	ImportantNotes []string           `yaml:"important_notes" json:"important_notes,omitempty"`
}

type StartingPoint struct {
	Provided       []string `yaml:"provided"        json:"provided"`
	LearnerChanges []string `yaml:"learner_changes" json:"learner_changes"`
	NotRequired    []string `yaml:"not_required"    json:"not_required"`
}

type OrientationStep struct {
	Title       string `yaml:"title"       json:"title"`
	Description string `yaml:"description" json:"description"`
	Purpose     string `yaml:"purpose"     json:"purpose"`
}

type OrientationSuccess struct {
	Criteria   []string `yaml:"criteria"    json:"criteria"`
	FinalState string   `yaml:"final_state" json:"final_state"`
}

type Scenario struct {
	ID            string        `yaml:"-"            json:"id"`
	Title         string        `yaml:"title"        json:"title"`
	Description   string        `yaml:"description"   json:"description"`
	Outcome       string        `yaml:"outcome"       json:"outcome"`
	Prerequisites []string      `yaml:"prerequisites" json:"prerequisites"`
	Duration      int           `yaml:"duration"      json:"duration"`
	Difficulty    string        `yaml:"difficulty"   json:"difficulty"`
	Tags          []string      `yaml:"tags"         json:"tags"`
	Orientation   Orientation   `yaml:"orientation"   json:"orientation"`
	Steps         []Step        `yaml:"steps"        json:"steps"`
	Browser       bool          `yaml:"browser"      json:"browser"`
	Playground    bool          `yaml:"playground"   json:"playground"`
	Image         string        `yaml:"image"        json:"image,omitempty"`
	Size          string        `yaml:"size"         json:"size,omitempty"`
	Services      []ServiceLink `yaml:"services"     json:"services"`
	Preview       *Preview      `yaml:"preview"      json:"-"`
}

type Summary struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Outcome       string   `json:"outcome"`
	Prerequisites []string `json:"prerequisites"`
	Duration      int      `json:"duration"`
	Difficulty    string   `json:"difficulty"`
	Playground    bool     `json:"playground"`
}

func Load(fsys fs.FS, id string) (*Scenario, error) {
	data, err := fs.ReadFile(fsys, id+"/scenario.yaml")
	if err != nil {
		return nil, fmt.Errorf("scenario %q not found: %w", id, err)
	}

	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario.yaml: %w", err)
	}
	s.ID = id
	if err := s.Orientation.validate(len(s.Steps)); err != nil {
		return nil, fmt.Errorf("scenario %q orientation: %w", id, err)
	}

	for i, step := range s.Steps {
		content, err := fs.ReadFile(fsys, id+"/"+step.Text)
		if err != nil {
			return nil, fmt.Errorf("step %d text file %q: %w", i+1, step.Text, err)
		}
		s.Steps[i].Content = string(content)
		s.Steps[i].HasVerify = step.Verify != ""
	}

	return &s, nil
}

func (o Orientation) validate(stepCount int) error {
	if o.Mission == "" {
		return fmt.Errorf("mission is required")
	}
	if len(o.StartingPoint.Provided) == 0 {
		return fmt.Errorf("starting_point.provided must not be empty")
	}
	if len(o.StartingPoint.LearnerChanges) == 0 && len(o.StartingPoint.NotRequired) == 0 {
		return fmt.Errorf("starting_point must define learner_changes or not_required")
	}
	if len(o.Journey) != stepCount {
		return fmt.Errorf("journey must contain one entry per step (got %d, want %d)", len(o.Journey), stepCount)
	}
	for i, step := range o.Journey {
		if step.Title == "" || step.Description == "" || step.Purpose == "" {
			return fmt.Errorf("journey entry %d requires title, description, and purpose", i+1)
		}
	}
	if len(o.Success.Criteria) == 0 || o.Success.FinalState == "" {
		return fmt.Errorf("success requires criteria and final_state")
	}
	if len(o.Tools) == 0 {
		return fmt.Errorf("tools must not be empty")
	}
	return nil
}

func ListSummaries(fsys fs.FS) ([]Summary, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	var out []Summary
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		s, err := Load(fsys, e.Name())
		if err != nil {
			continue
		}
		out = append(out, Summary{
			ID:            s.ID,
			Title:         s.Title,
			Description:   s.Description,
			Outcome:       s.Outcome,
			Prerequisites: s.Prerequisites,
			Duration:      s.Duration,
			Difficulty:    s.Difficulty,
			Playground:    s.Playground,
		})
	}
	return out, nil
}
