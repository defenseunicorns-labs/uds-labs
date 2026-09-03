// Package cloudinit renders the VM bootstrap user-data for a scenario.
//
// The template powers the KubeVirt cloudInitNoCloud volume — it embeds the
// scenario setup.sh, its verify scripts, and lab-inject.py as heredocs (ADR-0011).
// Extracted from session.Manager so the operator can render it without the server
// binary.
package cloudinit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template" // nosemgrep: go.lang.security.audit.xss.import-text-template.import-text-template -- renders cloud-init YAML/shell, not HTML
)

// Input is the data the user-data template expects.
type Input struct {
	SetupSh        string
	VerifyScripts  map[string]string
	AppFiles       map[string]string
	BrowserEnabled bool
	InjectPy       string
}

// Render reads a scenario's setup.sh and verify/* from scenariosFS and executes
// tmpl, returning the rendered bootstrap script. injectPy is the lab-inject.py
// source to embed; browserEnabled toggles the desktop/noVNC services.
func Render(tmpl *template.Template, scenariosFS fs.FS, scenarioID, injectPy string, browserEnabled bool) (string, error) {
	setupSh, err := fs.ReadFile(scenariosFS, scenarioID+"/setup.sh")
	if err != nil {
		return "", fmt.Errorf("scenario %q setup.sh: %w", scenarioID, err)
	}

	verify := map[string]string{}
	if entries, err := fs.ReadDir(scenariosFS, scenarioID+"/verify"); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			b, err := fs.ReadFile(scenariosFS, scenarioID+"/verify/"+e.Name())
			if err == nil {
				verify[e.Name()] = string(b)
			}
		}
	}

	appFiles, err := readAppFiles(scenariosFS, scenarioID)
	if err != nil {
		return "", fmt.Errorf("scenario %q app files: %w", scenarioID, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, Input{
		SetupSh:        string(setupSh),
		VerifyScripts:  verify,
		AppFiles:       appFiles,
		BrowserEnabled: browserEnabled,
		InjectPy:       injectPy,
	}); err != nil {
		return "", fmt.Errorf("render user-data for %q: %w", scenarioID, err)
	}
	return buf.String(), nil
}

func readAppFiles(scenariosFS fs.FS, scenarioID string) (map[string]string, error) {
	root := path.Join(scenarioID, "app")
	if _, err := fs.Stat(scenariosFS, root); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	files := map[string]string{}
	err := fs.WalkDir(scenariosFS, root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("app file %q is not regular", filePath)
		}
		relative := strings.TrimPrefix(filePath, root+"/")
		if relative == filePath || relative == "" || path.IsAbs(relative) || strings.HasPrefix(path.Clean(relative), "../") {
			return fmt.Errorf("invalid app file path %q", relative)
		}
		contents, err := fs.ReadFile(scenariosFS, filePath)
		if err != nil {
			return err
		}
		files[relative] = string(contents)
		return nil
	})
	return files, err
}
