package main

// Tenant lookup run example:
//
//   TENANT_BINDINGS=alice=1,bob=2 go run ./tenant-lookup
//
// It listens on TENANT_LOOKUP_LISTEN_ADDR, default :8081.
//
// API:
//   GET /lookup?tenant=alice
//     Returns {"tenant":"alice","env":"1","found":true}
//
//   POST /bind
//     JSON: {"tenant":"alice","env":"1"}
//
// This service simulates the SEaaS / tenant platform. In a real system the
// boundary service would query by account id, tenant id, employee id, or another
// existing business identifier. This demo uses "tenant" as that identifier.

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type tenantStore struct {
	mu       sync.RWMutex
	bindings map[string]string
}

type bindRequest struct {
	Tenant string `json:"tenant"`
	Env    string `json:"env"`
}

type lookupResponse struct {
	Tenant string `json:"tenant"`
	Env    string `json:"env"`
	Found  bool   `json:"found"`
}

func main() {
	addr := env("TENANT_LOOKUP_LISTEN_ADDR", ":8081")
	store := &tenantStore{bindings: parseBindings(os.Getenv("TENANT_BINDINGS"))}

	mux := http.NewServeMux()
	mux.HandleFunc("/lookup", store.handleLookup)
	mux.HandleFunc("/bind", store.handleBind)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("tenant-lookup listening on %s bindings=%d", addr, len(store.bindings))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (s *tenantStore) handleLookup(w http.ResponseWriter, req *http.Request) {
	tenant := strings.TrimSpace(req.URL.Query().Get("tenant"))
	if tenant == "" {
		http.Error(w, "tenant is required", http.StatusBadRequest)
		return
	}

	s.mu.RLock()
	env, ok := s.bindings[tenant]
	s.mu.RUnlock()
	if !ok {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	log.Printf("lookup tenant=%s env=%s", tenant, env)
	writeJSON(w, lookupResponse{Tenant: tenant, Env: env, Found: true})
}

func (s *tenantStore) handleBind(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var in bindRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	in.Tenant = strings.TrimSpace(in.Tenant)
	in.Env = strings.TrimSpace(in.Env)
	if in.Tenant == "" || in.Env == "" {
		http.Error(w, "tenant and env are required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.bindings[in.Tenant] = in.Env
	s.mu.Unlock()

	log.Printf("bound tenant=%s env=%s", in.Tenant, in.Env)
	w.WriteHeader(http.StatusNoContent)
}

func parseBindings(raw string) map[string]string {
	out := make(map[string]string)
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		tenant, env, ok := strings.Cut(item, "=")
		if !ok {
			log.Printf("ignore invalid tenant binding %q", item)
			continue
		}
		tenant = strings.TrimSpace(tenant)
		env = strings.TrimSpace(env)
		if tenant == "" || env == "" {
			log.Printf("ignore invalid tenant binding %q", item)
			continue
		}
		out[tenant] = env
	}
	return out
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
