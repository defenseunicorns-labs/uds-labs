// Copyright 2026 Defense Unicorns
// SPDX-License-Identifier: AGPL-3.0-or-later OR LicenseRef-Defense-Unicorns-Commercial

// scenario-preview serves a cluster-free, container-isolated scenario workspace.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	labplatform "github.com/defenseunicorns-labs/uds-labs"
	"github.com/defenseunicorns-labs/uds-labs/internal/preview"
	"github.com/defenseunicorns-labs/uds-labs/internal/proxy"
	"github.com/defenseunicorns-labs/uds-labs/internal/scenario"
)

//go:embed static
var staticFiles embed.FS

type server struct {
	scenario  *scenario.Scenario
	workspace string
}

type commandRequest struct {
	Command string `json:"command"`
}

type commandResponse struct {
	Output  string `json:"output"`
	Success bool   `json:"success"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "validate":
		validate(os.Args[2:])
	case "prepare":
		prepare(os.Args[2:])
	case "serve":
		serve(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: scenario-preview <validate|prepare|serve|verify> [flags]")
}

func validate(args []string) {
	flags := flag.NewFlagSet("validate", flag.ExitOnError)
	scenariosPath := flags.String("scenarios", "/scenarios", "scenario source directory")
	_ = flags.Parse(args)
	fsys := os.DirFS(*scenariosPath)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		log.Fatalf("read scenarios: %v", err)
	}
	var failures []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		s, err := scenario.Load(fsys, entry.Name())
		if err == nil {
			err = preview.Validate(fsys, s)
		}
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		fmt.Printf("ok %s\n", entry.Name())
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintln(os.Stderr, failure)
		}
		os.Exit(1)
	}
}

func prepare(args []string) {
	flags := flag.NewFlagSet("prepare", flag.ExitOnError)
	scenariosPath := flags.String("scenarios", "/scenarios", "scenario source directory")
	id := flags.String("scenario", "", "scenario id")
	workspace := flags.String("workspace", "", "writable scenario workspace")
	_ = flags.Parse(args)
	if *id == "" {
		log.Fatal("prepare requires --scenario")
	}
	fsys := os.DirFS(*scenariosPath)
	s, err := scenario.Load(fsys, *id)
	if err != nil {
		log.Fatal(err)
	}
	if *workspace == "" {
		*workspace = s.Preview.Workspace
	}
	if err := preview.Prepare(fsys, s, *workspace); err != nil {
		log.Fatal(err)
	}
}

func serve(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	scenariosPath := flags.String("scenarios", "/scenarios", "scenario source directory")
	id := flags.String("scenario", "python-to-uds", "scenario id")
	workspace := flags.String("workspace", "", "writable scenario workspace")
	listen := flags.String("listen", ":8080", "HTTP listen address")
	reset := flags.Bool("reset", true, "replace the workspace with the fixture before serving")
	_ = flags.Parse(args)

	fsys := os.DirFS(*scenariosPath)
	s, err := scenario.Load(fsys, *id)
	if err != nil {
		log.Fatalf("load scenario: %v", err)
	}
	if err := preview.Validate(fsys, s); err != nil {
		log.Fatalf("validate preview contract: %v", err)
	}
	if *workspace == "" {
		*workspace = s.Preview.Workspace
	}
	if *reset {
		if err := preview.Prepare(fsys, s, *workspace); err != nil {
			log.Fatalf("prepare workspace: %v", err)
		}
	}

	srv := &server{scenario: s, workspace: *workspace}
	labStatic, err := fs.Sub(labplatform.StaticFiles, "web/static")
	if err != nil {
		log.Fatalf("load shared UI styles: %v", err)
	}

	// ttyd is configured with /terminal as its base path, so preserve that
	// prefix while proxying its HTTP assets and WebSocket connection.
	terminal, err := proxy.Handler("http://127.0.0.1:7681", "")
	if err != nil {
		log.Fatalf("configure terminal proxy: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /terminal/", terminal)
	mux.HandleFunc("GET /style.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, labStatic, "style.css")
	})
	mux.HandleFunc("GET /api/scenario", srv.getScenario)
	mux.HandleFunc("GET /api/tree", srv.getTree)
	mux.HandleFunc("GET /api/files", srv.getFile)
	mux.HandleFunc("PUT /api/files", srv.putFile)
	mux.HandleFunc("POST /api/run", srv.runCommand)
	mux.HandleFunc("POST /api/verify/{step}", srv.verifyStep)
	mux.Handle("GET /", http.FileServerFS(mustSub(staticFiles, "static")))

	log.Printf("Scenario Preview: http://localhost%s", *listen)
	if err := http.ListenAndServe(*listen, mux); err != nil {
		log.Fatal(err)
	}
}

func verify(args []string) {
	flags := flag.NewFlagSet("verify", flag.ExitOnError)
	scenariosPath := flags.String("scenarios", "/scenarios", "scenario source directory")
	id := flags.String("scenario", "", "scenario id")
	workspace := flags.String("workspace", "", "scenario workspace")
	step := flags.Int("step", 0, "step number; zero verifies all checks")
	_ = flags.Parse(args)
	if *id == "" {
		log.Fatal("verify requires --scenario")
	}
	s, err := scenario.Load(os.DirFS(*scenariosPath), *id)
	if err != nil {
		log.Fatal(err)
	}
	var result preview.Result
	if *step == 0 {
		result = preview.VerifyAll(s, *workspace)
	} else {
		result = preview.Verify(s, *workspace, *step)
	}
	for _, message := range result.Messages {
		fmt.Println(message)
	}
	if !result.Passed {
		os.Exit(1)
	}
}

func (s *server) getScenario(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.scenario)
}

func (s *server) getTree(w http.ResponseWriter, _ *http.Request) {
	var files []string
	err := filepath.WalkDir(s.workspace, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".scenario-preview") {
			return nil
		}
		rel, err := filepath.Rel(s.workspace, filePath)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		http.Error(w, "read workspace", http.StatusInternalServerError)
		return
	}
	sort.Strings(files)
	writeJSON(w, files)
}

func (s *server) getFile(w http.ResponseWriter, r *http.Request) {
	filePath, err := s.workspaceFile(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	contents, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(contents)
}

func (s *server) putFile(w http.ResponseWriter, r *http.Request) {
	filePath, err := s.workspaceFile(r.URL.Query().Get("path"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	contents, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 256*1024))
	if err != nil {
		http.Error(w, "read file", http.StatusBadRequest)
		return
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		http.Error(w, "create directory", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(filePath, contents, 0o644); err != nil {
		http.Error(w, "write file", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) runCommand(w http.ResponseWriter, r *http.Request) {
	var request commandRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32*1024)).Decode(&request); err != nil || strings.TrimSpace(request.Command) == "" {
		http.Error(w, "command is required", http.StatusBadRequest)
		return
	}

	// The visible ttyd client and click-to-run blocks share this tmux session.
	// Loading the command as a paste buffer preserves multi-line heredocs and
	// makes the command appear in the learner's actual interactive terminal.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	load := exec.CommandContext(ctx, "tmux", "load-buffer", "-b", "scenario-preview-command", "-")
	load.Stdin = strings.NewReader(request.Command)
	if output, err := load.CombinedOutput(); err != nil {
		writeJSON(w, commandResponse{Output: fmt.Sprintf("send command: %v: %s", err, output), Success: false})
		return
	}
	for _, args := range [][]string{
		{"paste-buffer", "-d", "-b", "scenario-preview-command", "-t", "preview"},
		{"send-keys", "-t", "preview", "Enter"},
	} {
		if output, err := exec.CommandContext(ctx, "tmux", args...).CombinedOutput(); err != nil {
			writeJSON(w, commandResponse{Output: fmt.Sprintf("send command: %v: %s", err, output), Success: false})
			return
		}
	}
	writeJSON(w, commandResponse{Output: "Sent to Lab Terminal.", Success: true})
}

func (s *server) verifyStep(w http.ResponseWriter, r *http.Request) {
	var step int
	if _, err := fmt.Sscanf(r.PathValue("step"), "%d", &step); err != nil || step < 1 || step > len(s.scenario.Steps) {
		http.Error(w, "invalid step", http.StatusBadRequest)
		return
	}
	writeJSON(w, preview.Verify(s.scenario, s.workspace, step))
}

func (s *server) workspaceFile(requested string) (string, error) {
	if requested == "" {
		return "", errors.New("path is required")
	}
	clean := filepath.Clean(requested)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("path must stay inside the workspace")
	}
	return filepath.Join(s.workspace, clean), nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON: %v", err)
	}
}

func mustSub(fsys fs.FS, directory string) fs.FS {
	sub, err := fs.Sub(fsys, directory)
	if err != nil {
		panic(err)
	}
	return sub
}
