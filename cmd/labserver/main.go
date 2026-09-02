package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	labplatform "github.com/defenseunicorns-labs/uds-labs"
	labv1 "github.com/defenseunicorns-labs/uds-labs/api/v1alpha1"
	"github.com/defenseunicorns-labs/uds-labs/internal/buildinfo"
	"github.com/defenseunicorns-labs/uds-labs/internal/proxy"
	"github.com/defenseunicorns-labs/uds-labs/internal/scenario"
	"github.com/defenseunicorns-labs/uds-labs/internal/session"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type customerInfo struct {
	DisplayName string `yaml:"display_name"`
	CSE         string `yaml:"cse"`
}

type labConfig struct {
	AEGroup   string                  `yaml:"ae_group"`
	Customers map[string]customerInfo `yaml:"customers"`
}

type server struct {
	mgr         *session.Manager
	k8sClient   client.Client
	scenariosFS fs.FS
	staticFS    fs.FS
	ttlMinutes  int
	customers   map[string]customerInfo
	aeGroup     string
	devMode     bool // true only when DEV_MODE=true is explicitly set
}

const (
	maxBodyBytes = 1 << 20 // 1 MiB — cap on forwarded VM request bodies
)

func main() {
	ttlMinutes, err := strconv.Atoi(envOr("SESSION_TTL_MINUTES", "60"))
	if err != nil {
		log.Fatalf("SESSION_TTL_MINUTES is not a valid integer: %v", err)
	}
	maxActiveSessions, err := strconv.Atoi(envOr("MAX_ACTIVE_SESSIONS", "1"))
	if err != nil || maxActiveSessions < 1 {
		log.Fatalf("MAX_ACTIVE_SESSIONS must be a positive integer")
	}
	vmNamespace := envOr("VM_NAMESPACE", "uds-labs-vms")
	port := envOr("PORT", "8080")

	devMode := strings.EqualFold(os.Getenv("DEV_MODE"), "true")
	if devMode {
		log.Printf("WARNING: DEV_MODE=true — all authenticated users have admin access; do not run in production")
	}

	// Scenarios FS: OS override for development, embedded otherwise.
	var scenariosFS fs.FS
	if dir := os.Getenv("SCENARIOS_DIR"); dir != "" {
		scenariosFS = os.DirFS(dir)
		log.Printf("using scenarios from %s", dir)
	} else {
		sub, err := fs.Sub(labplatform.ScenariosFS, "scenarios")
		if err != nil {
			log.Fatalf("embedded scenarios: %v", err)
		}
		scenariosFS = sub
	}

	// Static files FS: OS override for development.
	var staticFS fs.FS
	if dir := os.Getenv("STATIC_DIR"); dir != "" {
		staticFS = os.DirFS(dir)
		log.Printf("using static files from %s", dir)
	} else {
		sub, err := fs.Sub(labplatform.StaticFiles, "web/static")
		if err != nil {
			log.Fatalf("embedded static: %v", err)
		}
		staticFS = sub
	}

	// Build k8s client (in-cluster or from KUBECONFIG for local dev).
	scheme := runtime.NewScheme()
	if err := labv1.AddToScheme(scheme); err != nil {
		log.Fatalf("register LabSession scheme: %v", err)
	}
	cfg, err := ctrl.GetConfig()
	if err != nil {
		log.Fatalf("get kubeconfig: %v", err)
	}
	k8s, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		log.Fatalf("build k8s client: %v", err)
	}

	mgr := session.NewManager(k8s, vmNamespace, time.Duration(ttlMinutes)*time.Minute, scenariosFS, maxActiveSessions)

	labCfg := loadConfig()
	srv := &server{
		mgr:         mgr,
		k8sClient:   k8s,
		scenariosFS: scenariosFS,
		staticFS:    staticFS,
		ttlMinutes:  ttlMinutes,
		customers:   labCfg.Customers,
		aeGroup:     labCfg.AEGroup,
		devMode:     devMode,
	}

	mux := http.NewServeMux()

	// Health check (always public)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		jsonOK(w, map[string]string{"version": buildinfo.Version})
	})

	// Protected API routes
	mux.HandleFunc("GET /api/config", srv.getConfig)
	mux.HandleFunc("GET /api/catalog", srv.getCatalog)
	mux.HandleFunc("GET /api/scenarios", srv.listScenarios)
	mux.HandleFunc("GET /api/scenarios/{id}", srv.getScenario)
	mux.HandleFunc("POST /api/sessions", srv.createSession)
	mux.HandleFunc("GET /api/sessions/me", srv.getMySession)
	mux.HandleFunc("GET /api/sessions/{id}", srv.getSession)
	mux.HandleFunc("DELETE /api/sessions/{id}", srv.deleteSession)
	mux.HandleFunc("POST /t/{id}/cmd", srv.injectCmd)
	mux.HandleFunc("POST /t/{id}/navigate", srv.navigateBrowser)
	mux.HandleFunc("POST /api/sessions/{id}/verify/{step}", srv.verifyStep)
	mux.HandleFunc("GET /api/sessions/{id}/services", srv.sessionServices)
	mux.HandleFunc("POST /api/sessions/{id}/pause", srv.pauseSession)
	mux.HandleFunc("POST /api/sessions/{id}/resume", srv.resumeSession)
	mux.HandleFunc("/t/{id}/", srv.terminalProxy)
	mux.HandleFunc("/t/{id}/shell/", srv.shellProxy)
	mux.HandleFunc("/vnc/{id}/", srv.browserProxy)
	mux.HandleFunc("GET /ide/{id}", srv.idePage)
	mux.HandleFunc("/ide-api/{id}/", srv.ideFileProxy)

	// Admin routes — protected by group membership in the signed JWT.
	mux.HandleFunc("GET /api/admin/sessions", srv.requireAE(srv.adminListSessions))
	mux.HandleFunc("DELETE /api/admin/sessions/{id}", srv.requireAE(srv.adminDeleteSession))
	mux.HandleFunc("GET /api/admin/csm", srv.requireAE(srv.adminCSM))
	mux.HandleFunc("GET /admin", srv.requireAE(srv.adminPage))
	mux.HandleFunc("GET /admin/csm", srv.requireAE(srv.csmDashboard))

	// Static file server (catch-all)
	mux.Handle("/", http.FileServerFS(staticFS))

	log.Printf("labserver listening on :%s (vm-namespace=%s)", port, vmNamespace)
	log.Fatal(http.ListenAndServe(":"+port, mux)) // nosemgrep: go.lang.security.audit.net.use-tls.use-tls -- TLS terminated by Istio mTLS sidecar; app-layer TLS would break the mesh
}

// ── Admin handlers ────────────────────────────────────────────────────────────

func (s *server) adminPage(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, s.staticFS, "admin.html")
}

func (s *server) csmDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFileFS(w, r, s.staticFS, "csm.html")
}

func (s *server) adminListSessions(w http.ResponseWriter, r *http.Request) {
	all := s.mgr.All(r.Context())

	type adminSession struct {
		SessionID  string    `json:"session_id"`
		UserEmail  string    `json:"user_email"`
		Scenario   string    `json:"scenario"`
		ServiceDNS string    `json:"service_dns"`
		Status     string    `json:"status"`
		ExpiresAt  time.Time `json:"expires_at"`
	}

	result := make([]adminSession, 0, len(all))
	for _, sess := range all {
		result = append(result, adminSession{
			SessionID:  sess.ID,
			UserEmail:  sess.UserEmail,
			Scenario:   sess.Scenario,
			ServiceDNS: sess.ServiceDNS,
			Status:     string(sess.Status),
			ExpiresAt:  sess.ExpiresAt,
		})
	}
	jsonOK(w, result)
}

func (s *server) adminDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	// Expire rather than hard-delete so the CSM dashboard retains history.
	if err := s.mgr.Expire(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) adminCSM(w http.ResponseWriter, r *http.Request) {
	type csmStep struct {
		Status      string     `json:"status"`
		StartedAt   *time.Time `json:"startedAt,omitempty"`
		CompletedAt *time.Time `json:"completedAt,omitempty"`
		DurationH   float64    `json:"durationH,omitempty"`
	}
	type csmUser struct {
		Email     string `json:"email"`
		Completed int    `json:"completed"`
		Active    bool   `json:"active"`
	}
	type csmCustomer struct {
		ID         string    `json:"id"`
		Name       string    `json:"name"`
		Scenario   string    `json:"scenario"`
		CSE        string    `json:"cse"`
		UserCount  int       `json:"user_count"`
		Users      []csmUser `json:"users"`
		StepTitles []string  `json:"step_titles"`
		Steps      []csmStep `json:"steps"`
	}

	type groupKey struct{ domain, scenario string }
	type group struct {
		domain string
		best   *session.Session
		users  []csmUser
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	groups := map[groupKey]*group{}
	for _, sess := range s.mgr.All(r.Context()) {
		// Skip sessions that expired more than 30 days ago from the dashboard.
		if sess.Status == session.StatusExpired && sess.ExpiresAt.Before(cutoff) {
			continue
		}
		domain := emailDomain(sess.UserEmail)
		if domain == "" {
			continue
		}
		k := groupKey{domain, sess.Scenario}
		g, ok := groups[k]
		if !ok {
			g = &group{domain: domain}
			groups[k] = g
		}
		isActive := sess.Status != session.StatusExpired
		g.users = append(g.users, csmUser{
			Email:     sess.UserEmail,
			Completed: len(sess.CompletedSteps),
			Active:    isActive,
		})
		// Best = most completed steps; tie-break: active beats expired.
		if g.best == nil ||
			len(sess.CompletedSteps) > len(g.best.CompletedSteps) ||
			(len(sess.CompletedSteps) == len(g.best.CompletedSteps) &&
				isActive && g.best.Status == session.StatusExpired) {
			g.best = sess
		}
	}

	result := make([]csmCustomer, 0, len(groups))
	for k, g := range groups {
		sess := g.best
		sc, err := scenario.Load(s.scenariosFS, k.scenario)
		if err != nil {
			continue
		}

		completed := len(sess.CompletedSteps)
		isActive := sess.Status != session.StatusExpired

		steps := make([]csmStep, len(sc.Steps))
		for i := range sc.Steps {
			switch {
			case i < completed:
				rec := sess.CompletedSteps[i]
				t := rec.CompletedAt
				prev := sess.CreatedAt
				if i > 0 {
					prev = sess.CompletedSteps[i-1].CompletedAt
				}
				steps[i] = csmStep{
					Status:      "passed",
					CompletedAt: &t,
					DurationH:   rec.CompletedAt.Sub(prev).Hours(),
				}
			case i == completed && isActive:
				var t time.Time
				if completed > 0 {
					t = sess.CompletedSteps[completed-1].CompletedAt
				} else {
					t = sess.CreatedAt
				}
				steps[i] = csmStep{Status: "active", StartedAt: &t}
			default:
				steps[i] = csmStep{Status: "pending"}
			}
		}

		titles := make([]string, len(sc.Steps))
		for i, step := range sc.Steps {
			titles[i] = step.Title
		}

		info := s.customers[g.domain]
		name := info.DisplayName
		if name == "" {
			name = g.domain
		}
		result = append(result, csmCustomer{
			ID:         g.domain,
			Name:       name,
			Scenario:   sc.Title,
			CSE:        info.CSE,
			UserCount:  len(g.users),
			Users:      g.users,
			StepTitles: titles,
			Steps:      steps,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	jsonOK(w, result)
}

func loadConfig() labConfig {
	data, err := labplatform.ConfigFiles.ReadFile("config/customers.yaml")
	if err != nil {
		log.Printf("customers.yaml not found, CSE/AE config disabled: %v", err)
		return labConfig{Customers: map[string]customerInfo{}}
	}
	var cfg labConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("customers.yaml parse error: %v", err)
		return labConfig{Customers: map[string]customerInfo{}}
	}
	if cfg.Customers == nil {
		cfg.Customers = map[string]customerInfo{}
	}
	if cfg.AEGroup == "" {
		cfg.AEGroup = "/UDS Core/Admin"
	}
	return cfg
}

func emailDomain(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}

// ── Existing handlers ─────────────────────────────────────────────────────────

func (s *server) getConfig(w http.ResponseWriter, r *http.Request) {
	jsonOK(w, map[string]any{
		"session_ttl_minutes": s.ttlMinutes,
		"is_ae":               s.isAE(groupsFromRequest(r), authedEmail(r)),
	})
}

func (s *server) getCatalog(w http.ResponseWriter, r *http.Request) {
	catalog, err := scenario.LoadCatalog(s.scenariosFS)
	if err != nil {
		log.Printf("load catalog: %v", err)
		jsonError(w, "cannot read catalog", http.StatusInternalServerError)
		return
	}
	jsonOK(w, catalog)
}

func (s *server) listScenarios(w http.ResponseWriter, r *http.Request) {
	summaries, err := scenario.ListSummaries(s.scenariosFS)
	if err != nil {
		jsonError(w, "cannot read scenarios", http.StatusInternalServerError)
		return
	}
	jsonOK(w, summaries)
}

func (s *server) getScenario(w http.ResponseWriter, r *http.Request) {
	sc, err := scenario.Load(s.scenariosFS, r.PathValue("id"))
	if err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}
	jsonOK(w, sc)
}

func (s *server) createSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scenario string `json:"scenario"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Scenario == "" {
		jsonError(w, "scenario required", http.StatusBadRequest)
		return
	}

	userEmail := authedEmail(r)
	cid := clientID(w, r)
	sess, err := s.mgr.Create(r.Context(), cid, req.Scenario, userEmail)
	if err != nil {
		if errors.Is(err, session.ErrSessionExists) {
			jsonError(w, "you already have an active lab session", http.StatusConflict)
			return
		}
		if errors.Is(err, session.ErrCapacityReached) {
			jsonError(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		log.Printf("create session: %v", err)
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(sess)
}

func (s *server) getMySession(w http.ResponseWriter, r *http.Request) {
	cid := clientID(w, r)
	sess, ok := s.mgr.GetActive(r.Context(), cid)
	if !ok {
		jsonError(w, "no active session", http.StatusNotFound)
		return
	}
	jsonOK(w, sess)
}

func (s *server) getSession(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.mgr.Get(r.Context(), r.PathValue("id"))
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	jsonOK(w, sess)
}

func (s *server) deleteSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	// Expire rather than hard-delete so the CSM dashboard retains history.
	if err := s.mgr.Expire(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) pauseSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := s.mgr.Pause(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) resumeSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := s.mgr.Resume(r.Context(), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) verifyStep(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.mgr.Get(r.Context(), r.PathValue("id"))
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}
	if sess.Status != session.StatusReady || sess.ServiceDNS == "" {
		jsonError(w, "terminal not ready", http.StatusServiceUnavailable)
		return
	}
	if !validServiceDNS(sess.ServiceDNS) {
		jsonError(w, "invalid session DNS", http.StatusInternalServerError)
		return
	}
	step := r.PathValue("step")
	body, _ := json.Marshal(map[string]string{"step": step})
	httpClient := &http.Client{Timeout: 35 * time.Second}
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s:7680/verify", sess.ServiceDNS),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		jsonError(w, "verify failed", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		jsonError(w, "verify: read response", http.StatusBadGateway)
		return
	}
	// Normalize: VM agents may return either "passed" or "pass".
	var result struct {
		Passed bool `json:"passed"`
		Pass   bool `json:"pass"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Printf("verify step %s: unexpected response from VM agent: %v", step, err)
	}
	passed := result.Passed || result.Pass
	if passed {
		if err := s.mgr.MarkStepComplete(r.Context(), r.PathValue("id"), step); err != nil {
			log.Printf("mark step complete: %v", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	normalized, _ := json.Marshal(map[string]bool{"passed": passed})
	_, _ = w.Write(normalized) // nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter -- server-generated JSON via json.Marshal, Content-Type: application/json
}

func (s *server) injectCmd(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.mgr.Get(r.Context(), r.PathValue("id"))
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if sess.Status != session.StatusReady || sess.ServiceDNS == "" {
		http.Error(w, "terminal not ready", http.StatusServiceUnavailable)
		return
	}
	if !validServiceDNS(sess.ServiceDNS) {
		http.Error(w, "invalid session DNS", http.StatusInternalServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s:7680/cmd", sess.ServiceDNS),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		http.Error(w, "injection failed", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	w.WriteHeader(resp.StatusCode)
}

func (s *server) navigateBrowser(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.mgr.Get(r.Context(), r.PathValue("id"))
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !sess.BrowserEnabled || sess.Status != session.StatusReady || sess.ServiceDNS == "" {
		http.Error(w, "browser not available", http.StatusServiceUnavailable)
		return
	}
	if !validServiceDNS(sess.ServiceDNS) {
		http.Error(w, "invalid session DNS", http.StatusInternalServerError)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Post(
		fmt.Sprintf("http://%s:7680/navigate", sess.ServiceDNS),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		http.Error(w, "navigate failed", http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	w.WriteHeader(resp.StatusCode)
}

func (s *server) sessionServices(w http.ResponseWriter, r *http.Request) {
	sess, ok := s.mgr.Get(r.Context(), r.PathValue("id"))
	if !ok {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		jsonError(w, "forbidden", http.StatusForbidden)
		return
	}

	sc, err := scenario.Load(s.scenariosFS, sess.Scenario)
	var services []scenario.ServiceLink
	if err == nil && len(sc.Services) > 0 {
		services = sc.Services
	}

	if sess.BrowserEnabled && sess.Status == session.StatusReady && sess.ServiceDNS != "" && validServiceDNS(sess.ServiceDNS) {
		httpClient := &http.Client{Timeout: 10 * time.Second}
		resp, err := httpClient.Get(fmt.Sprintf("http://%s:7680/services", sess.ServiceDNS))
		if err == nil {
			defer func() { _ = resp.Body.Close() }()
			var detected []scenario.ServiceLink
			if json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&detected) == nil {
				existing := make(map[string]bool, len(services))
				for _, svc := range services {
					existing[svc.URL] = true
				}
				for _, d := range detected {
					if !existing[d.URL] {
						services = append(services, d)
					}
				}
			}
		}
	}

	if services == nil {
		services = []scenario.ServiceLink{}
	}
	jsonOK(w, services)
}

func (s *server) browserProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !sess.BrowserEnabled {
		http.Error(w, "browser not available for this scenario", http.StatusNotFound)
		return
	}
	if sess.Status != session.StatusReady || sess.ServiceDNS == "" {
		http.Error(w, "terminal not ready", http.StatusServiceUnavailable)
		return
	}
	if !validServiceDNS(sess.ServiceDNS) {
		http.Error(w, "invalid session DNS", http.StatusInternalServerError)
		return
	}
	h, err := proxy.Handler(fmt.Sprintf("http://%s:6080", sess.ServiceDNS), fmt.Sprintf("/vnc/%s", id))
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}
	h.ServeHTTP(w, r)
}

func (s *server) idePage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.ServeFileFS(w, r, s.staticFS, "ide.html")
}

func (s *server) ideFileProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if sess.Status != session.StatusReady {
		http.Error(w, "IDE not ready", http.StatusServiceUnavailable)
		return
	}
	if !validServiceDNS(sess.ServiceDNS) {
		http.Error(w, "invalid session DNS", http.StatusInternalServerError)
		return
	}
	h, err := proxy.Handler(fmt.Sprintf("http://%s:7680", sess.ServiceDNS), fmt.Sprintf("/ide-api/%s", id))
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}
	h.ServeHTTP(w, r)
}

func (s *server) terminalProxy(w http.ResponseWriter, r *http.Request) {
	s.proxyToVM(w, r, r.PathValue("id"), 7681, fmt.Sprintf("/t/%s", r.PathValue("id")))
}

func (s *server) shellProxy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.proxyToVM(w, r, id, 7682, fmt.Sprintf("/t/%s/shell", id))
}

func (s *server) proxyToVM(w http.ResponseWriter, r *http.Request, id string, port int, stripPrefix string) {
	sess, ok := s.mgr.Get(r.Context(), id)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !ownsSession(r, sess) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// The setup-aware main terminal is intentionally available while the VM is
	// running so it can stream bootstrap output. Keep extra shells gated until
	// the lab is fully ready.
	if (port != 7681 && sess.Status != session.StatusReady) ||
		(port == 7681 && sess.Status != session.StatusRunning && sess.Status != session.StatusReady) ||
		sess.ServiceDNS == "" {
		http.Error(w, "terminal not ready", http.StatusServiceUnavailable)
		return
	}
	if !validServiceDNS(sess.ServiceDNS) {
		http.Error(w, "invalid session DNS", http.StatusInternalServerError)
		return
	}
	h, err := proxy.Handler(fmt.Sprintf("http://%s:%d", sess.ServiceDNS, port), stripPrefix)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}
	h.ServeHTTP(w, r)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// authedEmail returns the authenticated user's email from the signed JWT bearer
// token. UDS Core authservice sets Authorization: Bearer <id_token>; Istio's
// RequestAuthentication validates the signature before the request reaches the
// app. X-Auth-Request-Email is NOT read here — UDS Core authservice does not
// inject it, and accepting it would allow email spoofing for session attribution.
func authedEmail(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return emailFromJWT(strings.TrimPrefix(auth, "Bearer "))
	}
	return ""
}

// emailFromJWT extracts the email claim from a JWT without signature verification.
// Safe here because the token was already validated by Istio's RequestAuthentication.
func emailFromJWT(token string) string {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(claims.Email))
}

// groupsFromJWT extracts the groups claim from a JWT without signature verification.
// Safe for the same reason as emailFromJWT — Istio already validated the token.
// Keycloak emits groups as a []string with full paths (e.g. "/UDS Core/Admin").
func groupsFromJWT(token string) []string {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims struct {
		Groups []string `json:"groups"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims.Groups
}

// ownsSession checks whether the requesting user owns the session.
// In production, authservice injects the ID token as Authorization: Bearer and
// we parse the email from it. In dev (no authservice), we fall back to the UUID
// cookie so local testing still works.
// If an Authorization: Bearer header is present but the JWT carries no email
// (malformed or expired token), we do NOT fall back to the cookie — the request
// was explicitly trying to authenticate and we must not silently downgrade to
// cookie auth, which any browser on the same machine could satisfy.
// X-Auth-Request-Email is NOT checked here — UDS Core authservice does not
// inject it, and accepting it would allow session ownership spoofing.
func ownsSession(r *http.Request, sess *session.Session) bool {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		email := emailFromJWT(strings.TrimPrefix(auth, "Bearer "))
		if email == "" {
			return false
		}
		return strings.EqualFold(email, sess.UserEmail)
	}
	c, err := r.Cookie("lab_client_id")
	return err == nil && c.Value != "" && c.Value == sess.ClientID
}

// clientID returns a stable client identifier for the requesting user.
// In production, we derive a deterministic label-safe key from the Keycloak
// email so that one user always maps to one client regardless of browser or
// device. In dev (no authservice), we fall back to a UUID cookie.
func clientID(w http.ResponseWriter, r *http.Request) string {
	if email := authedEmail(r); email != "" {
		return emailClientID(email)
	}
	const cookieName = "lab_client_id"
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	id := uuid.New().String()
	http.SetCookie(w, &http.Cookie{ // nosemgrep: go.lang.security.audit.net.cookie-missing-secure.cookie-missing-secure -- Secure set dynamically via isSecureRequest
		Name:     cookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		Secure:   isSecureRequest(r),
		SameSite: http.SameSiteLaxMode,
	})
	return id
}

// emailClientID derives a Kubernetes label-safe client key from an email.
// Uses the first 32 hex chars of SHA-256(lower(email)).
func emailClientID(email string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email))))
	return hex.EncodeToString(h[:16])
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ── Admin authorization ────────────────────────────────────────────────────── ───────────────────────────────────────────────────────────

// groupsFromRequest resolves the caller's group list from the signed JWT bearer
// token. UDS Core authservice (istio-ecosystem/authservice) sets Authorization:
// Bearer <id_token>; Istio's RequestAuthentication validates the signature before
// the request reaches the app. We do NOT read X-Auth-Request-Groups because UDS
// Core authservice does not inject it, Istio does not strip client-supplied copies
// of it, and accepting it would allow any authenticated user to inject arbitrary
// group membership.
func groupsFromRequest(r *http.Request) []string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return groupsFromJWT(strings.TrimPrefix(auth, "Bearer "))
	}
	return nil
}

func (s *server) requireAE(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.isAE(groupsFromRequest(r), authedEmail(r)) {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

// isAE returns true if the request comes from an AE/admin user.
// groups is the resolved list from the signed JWT bearer token.
// In dev mode (DEV_MODE=true), any authenticated email suffices as a fallback
// when no groups are present — never enabled in production.
func (s *server) isAE(groups []string, email string) bool {
	if len(groups) > 0 {
		for _, g := range groups {
			if g == s.aeGroup {
				return true
			}
		}
		return false
	}
	// Dev only: allow any authenticated user when DEV_MODE is explicitly set.
	if s.devMode {
		return strings.TrimSpace(email) != ""
	}
	return false
}

// validServiceDNS returns true if the DNS name is a safe in-cluster service
// name: only alphanumerics, hyphens, and dots, no path/port/scheme characters
// that could be used for SSRF. The name must be non-empty.
func validServiceDNS(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '-' && c != '.' {
			return false
		}
	}
	return true
}

// isSafeHost returns true if h is a plain host[:port] with no scheme, path,
// or other characters that could redirect the generated URL to an attacker host.
func isSafeHost(h string) bool {
	if h == "" {
		return false
	}
	for _, c := range h {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') &&
			(c < '0' || c > '9') && c != '-' && c != '.' && c != ':' && c != '[' && c != ']' {
			return false
		}
	}
	return true
}

// isSecureRequest returns true when the connection is HTTPS, either directly
// or via a TLS-terminating proxy (X-Forwarded-Proto: https).
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https"
}
