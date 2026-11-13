package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()

	healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyz(t *testing.T) {
	tmpDir := t.TempDir() // a temp folder, auto-deleted when the test ends

	cases := []struct {
		name       string
		fileExists bool
		content    string
		wantStatus int
	}{
		{"missing file", false, "", http.StatusServiceUnavailable},
		{"empty file", true, "", http.StatusServiceUnavailable},
		{"valid key", true, "dev-key", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := tmpDir + "/" + tc.name + "-key"
			if tc.fileExists {
				os.WriteFile(path, []byte(tc.content), 0o600)
			}

			secretPath = path                                      // point the handler at this test's file
			defer func() { secretPath = "/mnt/secrets/api-key" }() // restore after

			req := httptest.NewRequest("GET", "/readyz", nil)
			rec := httptest.NewRecorder()
			readyz(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.wantStatus)
			}
		})
	}
}

func TestListTransactions(t *testing.T) {
	tmpDir := t.TempDir()
	path := tmpDir + "/api-key"
	os.WriteFile(path, []byte("dev-key"), 0o600)
	secretPath = path
	defer func() { secretPath = "/mnt/secrets/api-key" }()

	cases := []struct {
		name       string
		givenKey   string
		wantStatus int
	}{
		{"no key", "", http.StatusUnauthorized},
		{"wrong key", "nope", http.StatusUnauthorized},
		{"correct key", "dev-key", http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/transactions", nil)
			if tc.givenKey != "" {
				req.Header.Set("X-API-Key", tc.givenKey)
			}
			rec := httptest.NewRecorder()
			listTransactions(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("%s: got %d, want %d", tc.name, rec.Code, tc.wantStatus)
			}
		})
	}
}
