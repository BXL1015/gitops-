package main

// Registry run example:
//
//   go run ./registry
//
// Or after compiling:
//
//   ./registry/registry
//
// It listens on REGISTRY_LISTEN_ADDR, default :8080.
//
// API:
//   POST /register
//     JSON: {"name":"svc1","env":"0","addr":"http://127.0.0.1:9001"}
//
//   GET /lookup?name=svc2&env=1
//     Returns svc2 in env 1 if present; otherwise returns svc2 in env 0.

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

type registry struct {
	mu       sync.RWMutex
	services map[string]map[string]string
}

type registerRequest struct {
	Name string `json:"name"`
	Env  string `json:"env"`
	Addr string `json:"addr"`
}

type lookupResponse struct {
	Name      string `json:"name"`
	Requested string `json:"requested_env"`
	Env       string `json:"env"`
	Addr      string `json:"addr"`
	Fallback  bool   `json:"fallback"`
}

func main() {
	addr := env("REGISTRY_LISTEN_ADDR", ":8080")
	r := &registry{services: make(map[string]map[string]string)}

	mux := http.NewServeMux()
	mux.HandleFunc("/register", r.handleRegister)
	mux.HandleFunc("/lookup", r.handleLookup)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	s := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("registry listening on %s", addr)
	if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (r *registry) handleRegister(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var in registerRequest
	if err := json.NewDecoder(req.Body).Decode(&in); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Env = strings.TrimSpace(in.Env)
	in.Addr = strings.TrimSpace(in.Addr)
	if in.Name == "" || in.Env == "" || in.Addr == "" {
		http.Error(w, "name, env and addr are required", http.StatusBadRequest)
		return
	}

	r.mu.Lock()
	if r.services[in.Name] == nil {
		r.services[in.Name] = make(map[string]string)
	}
	r.services[in.Name][in.Env] = in.Addr
	r.mu.Unlock()

	log.Printf("registered name=%s env=%s addr=%s", in.Name, in.Env, in.Addr)
	w.WriteHeader(http.StatusNoContent)
}

func (r *registry) handleLookup(w http.ResponseWriter, req *http.Request) {
	name := strings.TrimSpace(req.URL.Query().Get("name"))
	targetEnv := strings.TrimSpace(req.URL.Query().Get("env"))
	if name == "" || targetEnv == "" {
		http.Error(w, "name and env are required", http.StatusBadRequest)
		return
	}

	env, addr, ok := r.lookup(name, targetEnv)
	if !ok {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}
	log.Printf("lookup name=%s requested_env=%s selected_env=%s fallback=%t addr=%s",
		name, targetEnv, env, env != targetEnv, addr)

	writeJSON(w, lookupResponse{
		Name:      name,
		Requested: targetEnv,
		Env:       env,
		Addr:      addr,
		Fallback:  env != targetEnv,
	})
}

func (r *registry) lookup(name, targetEnv string) (string, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	byEnv := r.services[name]
	if byEnv == nil {
		return "", "", false
	}
	if addr := byEnv[targetEnv]; addr != "" {
		return targetEnv, addr, true
	}
	if addr := byEnv["0"]; addr != "" {
		return "0", addr, true
	}
	return "", "", false
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
