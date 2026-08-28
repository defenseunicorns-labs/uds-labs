// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

// Package operator holds wiring for the Lab Operator binary: configuration
// loaded from the operator ConfigMap (ADR-0013) — size tiers and image refs.
package operator

import (
	"fmt"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"

	"github.com/defenseunicorns-labs/uds-labs/internal/sizing"
)

// Config is the operator ConfigMap, mounted as a file (path in OPERATOR_CONFIG).
// It maps abstract sizes to resources and image tiers to golden PVC names.
// Golden PVCs are pre-populated qcow2 disks; CDI clones one per session.
type Config struct {
	Sizes              map[string]sizeEntry `yaml:"sizes"`
	GoldenPVCs         map[string]string    `yaml:"goldenPVCs"`         // tier → PVC name
	GoldenPVCNamespace string               `yaml:"goldenPVCNamespace"` // defaults to VM namespace
	GoldenPVCDiskSize  string               `yaml:"goldenPVCDiskSize"`  // usable clone size requested through CDI storage API
	StorageClass       string               `yaml:"storageClass"`
	VMIPlacement       Placement            `yaml:"vmiPlacement"`
	BlockedEgressCIDRs []string             `yaml:"blockedEgressCIDRs"`
}

// Placement is the portable Kubernetes scheduling contract for Lab VMIs.
type Placement struct {
	NodeSelector map[string]string   `yaml:"nodeSelector"`
	Affinity     *corev1.Affinity    `yaml:"affinity"`
	Tolerations  []corev1.Toleration `yaml:"tolerations"`
}

type sizeEntry struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

// SizeOverrides converts the configured sizes into the sizing package's shape.
// Unknown tier names are rejected so a typo in the ConfigMap fails loudly.
func (c *Config) SizeOverrides() (map[sizing.Size]sizing.Spec, error) {
	out := map[sizing.Size]sizing.Spec{}
	for name, e := range c.Sizes {
		s := sizing.Size(name)
		if !sizing.Valid(s) {
			return nil, fmt.Errorf("config sizes: unknown tier %q", name)
		}
		out[s] = sizing.Spec{CPU: e.CPU, Memory: e.Memory}
	}
	return out, nil
}

// Load reads the operator ConfigMap from path. A missing path yields an empty
// Config (built-in size defaults, no images), which is valid for startup but
// means sessions fail until images are configured.
func Load(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read operator config %q: %w", path, err)
	}
	return parse(data)
}

// LoadFS is Load against an fs.FS (used in tests).
func LoadFS(fsys fs.FS, path string) (*Config, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return parse(data)
}

func parse(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse operator config: %w", err)
	}
	return &c, nil
}
