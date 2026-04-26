package business

// Business service run example:
//
// 1. Start registry and tenant lookup:
//      go run ./registry
//      TENANT_BINDINGS=alice=1,bob=2 go run ./tenant-lookup
//
// 2. Start baseline services in env 0. By default svc1 calls svc2, svc2 calls
//    svc3, and so on until svc15.
//      SERVICE_ENV=0 LISTEN_ADDR=:9001 PUBLIC_ADDR=http://127.0.0.1:9001 go run ./svc1
//
// 3. Start hot services svc3~svc8 in env 1.
//      SERVICE_ENV=1 LISTEN_ADDR=:9103 PUBLIC_ADDR=http://127.0.0.1:9103 go run ./svc3
//
// 4. Call the baseline entrance with a business tenant/account id:
//      curl "http://127.0.0.1:9001/?tenant=alice&n=123"
//
// Routing rule:
//   Most services look up downstream with their own SERVICE_ENV.
//   Boundary services can query tenant-lookup with a business tenant/account id
//   and use that env for the next hop. By default svc2 is a boundary service,
//   because its downstream svc3 is the beginning of the hot SET layer.
//   No env id is propagated in the request; only the simulated business tenant
//   id is forwarded so boundary services can recover the SET env.

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
	name         string
	env          string
	listenAddr   string
	publicAddr   string
	registry     string
	downstream   string
	lookupMode   string
	tenantLookup string
	tenantHeader string
	client       *http.Client
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

type tenantLookupResponse struct {
	Tenant string `json:"tenant"`
	Env    string `json:"env"`
	Found  bool   `json:"found"`
}

// Run starts one business microservice. defaultName is used only when
// SERVICE_NAME is not set, so each svcN/main.go can still be run directly.
func Run(defaultName string) {
	debug.SetGCPercent(50)

	name := env("SERVICE_NAME", defaultName)
	s := &service{
		name:         name,
		env:          env("SERVICE_ENV", "0"),
		listenAddr:   env("LISTEN_ADDR", ":9000"),
		publicAddr:   env("PUBLIC_ADDR", "http://127.0.0.1:9000"),
		registry:     trimRightSlash(env("REGISTRY_ADDR", "http://127.0.0.1:8080")),
		downstream:   configuredDownstream(name),
		lookupMode:   env("LOOKUP_MODE", defaultLookupMode(name)),
		tenantLookup: trimRightSlash(env("TENANT_LOOKUP_ADDR", "http://127.0.0.1:8081")),
		tenantHeader: env("TENANT_HEADER", "X-Tenant"),
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

	log.Printf("service=%s env=%s listen=%s public=%s downstream=%s lookup_mode=%s",
		s.name, s.env, s.listenAddr, s.publicAddr, valueOrDash(s.downstream), s.lookupMode)

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

	tenant := s.tenantFrom(r)
	targetEnv := s.routeEnv(tenant)
	next, err := s.lookup(s.downstream, targetEnv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if next.Fallback {
		log.Printf("route service=%s env=%s tenant=%s downstream=%s requested_env=%s selected_env=%s fallback=true addr=%s",
			s.name, s.env, valueOrDash(tenant), s.downstream, next.Requested, next.Env, next.Addr)
	} else {
		log.Printf("route service=%s env=%s tenant=%s downstream=%s selected_env=%s addr=%s",
			s.name, s.env, valueOrDash(tenant), s.downstream, next.Env, next.Addr)
	}

	resp, err := s.callDownstream(next.Addr, n, tenant)
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

func (s *service) routeEnv(tenant string) string {
	if s.lookupMode == "none" || tenant == "" {
		return s.env
	}

	env, ok, err := s.lookupTenant(tenant)
	if err != nil {
		log.Printf("tenant lookup failed service=%s tenant=%s err=%v; use own env=%s",
			s.name, tenant, err, s.env)
		return s.env
	}
	if !ok || env == "" {
		log.Printf("tenant lookup miss service=%s tenant=%s; use own env=%s",
			s.name, tenant, s.env)
		return s.env
	}

	log.Printf("tenant lookup hit service=%s tenant=%s lookup_env=%s own_env=%s",
		s.name, tenant, env, s.env)
	return env
}

func (s *service) lookupTenant(tenant string) (string, bool, error) {
	u, err := url.Parse(s.tenantLookup + "/lookup")
	if err != nil {
		return "", false, err
	}
	q := u.Query()
	q.Set("tenant", tenant)
	u.RawQuery = q.Encode()

	resp, err := s.client.Get(u.String())
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", false, nil
	}
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", false, fmt.Errorf("tenant lookup returned %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}

	var out tenantLookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false, err
	}
	return out.Env, out.Found, nil
}

func (s *service) tenantFrom(r *http.Request) string {
	if tenant := strings.TrimSpace(r.URL.Query().Get("tenant")); tenant != "" {
		return tenant
	}
	if tenant := strings.TrimSpace(r.Header.Get(s.tenantHeader)); tenant != "" {
		return tenant
	}
	if cookie, err := r.Cookie("tenant"); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func (s *service) callDownstream(addr, n, tenant string) (*http.Response, error) {
	u, err := url.Parse(trimRightSlash(addr) + "/")
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("n", n)
	if tenant != "" {
		q.Set("tenant", tenant)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if tenant != "" {
		req.Header.Set(s.tenantHeader, tenant)
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

func configuredDownstream(name string) string {
	raw := strings.TrimSpace(os.Getenv("DOWNSTREAM_SERVICE"))
	if raw == "-" || strings.EqualFold(raw, "none") {
		return ""
	}
	if raw != "" {
		return raw
	}
	return defaultDownstream(name)
}

func defaultDownstream(name string) string {
	if !strings.HasPrefix(name, "svc") {
		return ""
	}
	var n int
	if _, err := fmt.Sscanf(name, "svc%d", &n); err != nil {
		return ""
	}
	if n < 1 || n >= 15 {
		return ""
	}
	return fmt.Sprintf("svc%d", n+1)
}

func defaultLookupMode(name string) string {
	if name == "svc2" {
		return "boundary"
	}
	return "none"
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
