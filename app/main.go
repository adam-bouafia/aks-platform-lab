package main

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

var secretPath = getenv("SECRET_PATH", "/mnt/secrets/api-key")

type Transaction struct {
	ID          string `json:"id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
}

var transactions = []Transaction{
	{ID: "tx-001", AmountCents: 12050, Currency: "EUR"},
	{ID: "tx-002", AmountCents: -4500, Currency: "EUR"},
}

func main() {
	// 1. Set up the JSON logger as the default
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. Read the port configuration
	port := getenv("PORT", "8080")

	// 3. Create the mux and register your first route
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /readyz", readyz)
	mux.HandleFunc("GET /api/v1/transactions", listTransactions)
	// 4. Configure the HTTP server with safe timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Start the server
	slog.Info("server starting", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}

// getenv reads an environment variable or returns a fallback if empty.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// healthz tells Kubernetes the process is alive.
func healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}

// readyz: 200 if the key file exists and is not empty, otherwise 503
func readyz(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(secretPath) // try to read the file
	if err != nil || strings.TrimSpace(string(data)) == "" {
		w.WriteHeader(http.StatusServiceUnavailable) // 503: no file, or empty
		fmt.Fprintln(w, "not ready")
		return
	}
	w.WriteHeader(http.StatusOK) // 200: key is there
	fmt.Fprintln(w, "ready")
}

func listTransactions(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(secretPath) // the real key
	key := strings.TrimSpace(string(data))
	given := r.Header.Get("X-API-Key") // the key the caller sent

	if err != nil || key == "" || subtle.ConstantTimeCompare([]byte(given), []byte(key)) != 1 {
		w.WriteHeader(http.StatusUnauthorized) // 401
		fmt.Fprintln(w, "unauthorized")
		return
	}

	w.Header().Set("Content-Type", "application/json") // tell the caller it's JSON
	json.NewEncoder(w).Encode(transactions)            // turn the list into JSON and send it
}
