package main

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var secretPath = getenv("SECRET_PATH", "/mnt/secrets/api-key")
var errorRate, _ = strconv.ParseFloat(getenv("ERROR_RATE", "0"), 64)

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
	mux.HandleFunc("GET /api/v1/transactions", withLogging("/api/v1/transactions", listTransactions))
	mux.Handle("GET /metrics", promhttp.Handler())
	// 4. Configure the HTTP server with safe timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 5. Start the server in the background
	slog.Info("server starting", "addr", srv.Addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// 6. Wait here until Kubernetes (or Ctrl+C) asks us to stop
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-ctx.Done()

	// 7. Shut down gracefully, giving in-flight requests up to 10s to finish
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
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

func withLogging(route string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if rand.Float64() < errorRate {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintln(w, "injected error")
			slog.Error("request", "route", route, "status", 500, "duration_ms", time.Since(start).Milliseconds())
			return
		}

		h(w, r)
		slog.Info("request", "route", route, "duration_ms", time.Since(start).Milliseconds())
	}
}
