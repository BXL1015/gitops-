package business

// Business service run example:
//
// 1. Start registry:
//      go run ./registry
//
// 2. Start a baseline service in env 0:
//      SERVICE_NAME=svc3 SERVICE_ENV=0 LISTEN_ADDR=:9003 PUBLIC_ADDR=http://127.0.0.1:9003 REGISTRY_ADDR=http://127.0.0.1:8080 go run ./svc3
//
// 3. Start a lane service in env 1:
//      SERVICE_NAME=svc3 SERVICE_ENV=1 LISTEN_ADDR=:9103 PUBLIC_ADDR=http://127.0.0.1:9103 REGISTRY_ADDR=http://127.0.0.1:8080 go run ./svc3
//
// 4. Start an upstream service:
//      SERVICE_NAME=svc2 SERVICE_ENV=1 DOWNSTREAM_SERVICE=svc3 LISTEN_ADDR=:9102 PUBLIC_ADDR=http://127.0.0.1:9102 REGISTRY_ADDR=http://127.0.0.1:8080 go run ./svc2
//
// 5. Call it:
//      curl "http://127.0.0.1:9102/?n=123"
//
// Routing rule:
//   The request does not carry an env header or any other lane metadata.
//   Each service looks up its downstream with its own SERVICE_ENV.
//   If the registry falls back to env 0 at one hop, the next hop runs inside
//   an env 0 process, so the rest of the chain naturally keeps looking for env 0.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime/debug"
	"strings"
	"time"
)

type service struct {
	name       string
	env        string
	listenAddr string
	publicAddr string
	registry   string
	downstream string
	client     *http.Client
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

// Run starts one business microservice. defaultName is used only when
// SERVICE_NAME is not set, so each svcN/main.go can still be run directly.
func Run(defaultName string) {
	debug.SetGCPercent(50)

	s := &service{
		name:       env("SERVICE_NAME", defaultName),
		env:        env("SERVICE_ENV", "0"),
		listenAddr: env("LISTEN_ADDR", ":9000"),
		publicAddr: env("PUBLIC_ADDR", "http://127.0.0.1:9000"),
		registry:   trimRightSlash(env("REGISTRY_ADDR", "http://127.0.0.1:8080")),
		downstream: strings.TrimSpace(os.Getenv("DOWNSTREAM_SERVICE")),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
	}

	if err := s.register(); err != nil {
		log.Fatalf("register service: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handle)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{
		Addr:              s.listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("service=%s env=%s listen=%s public=%s downstream=%s",
		s.name, s.env, s.listenAddr, s.publicAddr, valueOrDash(s.downstream))

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func (s *service) handle(w http.ResponseWriter, r *http.Request) {
	n, err := readNumber(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.downstream == "" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Service-Name", s.name)
		w.Header().Set("X-Service-Env", s.env)
		_, _ = fmt.Fprintf(w, "service=%s env=%s n=%s\n", s.name, s.env, n)
		return
	}

	next, err := s.lookup(s.downstream, s.env)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if next.Fallback {
		log.Printf("route service=%s env=%s downstream=%s requested_env=%s selected_env=%s fallback=true addr=%s",
			s.name, s.env, s.downstream, next.Requested, next.Env, next.Addr)
	} else {
		log.Printf("route service=%s env=%s downstream=%s selected_env=%s addr=%s",
			s.name, s.env, s.downstream, next.Env, next.Addr)
	}

	resp, err := s.callDownstream(next.Addr, n)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyResponseHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func (s *service) register() error {
	body, err := json.Marshal(registerRequest{
		Name: s.name,
		Env:  s.env,
		Addr: s.publicAddr,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, s.registry+"/register", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("registry returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	return nil
}

func (s *service) lookup(name, targetEnv string) (*lookupResponse, error) {
	u, err := url.Parse(s.registry + "/lookup")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("name", name)
	q.Set("env", targetEnv)
	u.RawQuery = q.Encode()

	resp, err := s.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("lookup %s env %s failed: %s %s", name, targetEnv, resp.Status, strings.TrimSpace(string(msg)))
	}

	var out lookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) callDownstream(addr, n string) (*http.Response, error) {
	u, err := url.Parse(trimRightSlash(addr) + "/")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("n", n)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	return s.client.Do(req)
}

func readNumber(r *http.Request) (string, error) {
	if n := strings.TrimSpace(r.URL.Query().Get("n")); n != "" {
		return n, nil
	}

	if r.Body == nil {
		return "0", nil
	}
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		return "", err
	}
	n := strings.TrimSpace(string(body))
	if n == "" {
		return "0", nil
	}
	return n, nil
}

func copyResponseHeaders(dst, src http.Header) {
	for k, values := range src {
		if strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, v := range values {
			dst.Add(k, v)
		}
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func trimRightSlash(s string) string {
	return strings.TrimRight(s, "/")
}

func valueOrDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
