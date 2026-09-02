package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireAEAllows(t *testing.T) {
	srv := &server{aeGroup: "/UDS Core/Admin"}
	called := false
	h := srv.requireAE(func(w http.ResponseWriter, r *http.Request) { called = true })

	r := httptest.NewRequest("GET", "/api/admin/sessions", nil)
	r.Header.Set("X-Auth-Request-Groups", "/Other Group")
	r.Header.Set("Authorization", "Bearer "+testJWTWithGroups("/Other Group", "/UDS Core/Admin"))
	w := httptest.NewRecorder()
	h(w, r)

	if !called {
		t.Error("handler was not called for user in AE group")
	}
}

func TestRequireAEBlocks(t *testing.T) {
	srv := &server{aeGroup: "/UDS Core/Admin"}
	called := false
	h := srv.requireAE(func(w http.ResponseWriter, r *http.Request) { called = true })

	r := httptest.NewRequest("GET", "/api/admin/sessions", nil)
	r.Header.Set("X-Auth-Request-Groups", "/UDS Core/Admin")
	r.Header.Set("Authorization", "Bearer "+testJWTWithGroups("/Other Group", "/UDS Core/Administrators"))
	w := httptest.NewRecorder()
	h(w, r)

	if called {
		t.Error("handler should not be called for user NOT in AE group")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func testJWTWithGroups(groups ...string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payloadJSON, _ := json.Marshal(struct {
		Groups []string `json:"groups"`
	}{Groups: groups})
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payload + ".test-signature"
}

func TestIsAESubstringFalsePositive(t *testing.T) {
	srv := &server{aeGroup: "/UDS Core/Admin"}
	// "/UDS Core/Administrators" contains "/UDS Core/Admin" as substring —
	// but isAE must use exact match, so this should be false.
	if srv.isAE([]string{"/UDS Core/Administrators"}, "") {
		t.Error("isAE should not match substring — only exact group name")
	}
}

func TestIsAEEmptyGroup(t *testing.T) {
	srv := &server{aeGroup: ""}
	if srv.isAE([]string{"/anything"}, "") {
		t.Error("isAE with empty aeGroup should always return false")
	}
}
