package business

import (
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

// service �ṹ����ȫ���� Istio (Sidecar) ������
// �Ƴ��� registry �� tenant-lookup ����������Ϊ����Ⱦɫ��·���� Envoy Sidecar ��ɡ�
type service struct {
name         string
env          string
listenAddr   string
downstream   string
tenantHeader string
client       *http.Client
}

func Run(defaultName string) {
debug.SetGCPercent(50)

name := env("SERVICE_NAME", defaultName)
s := &service{
name:         name,
env:          env("SERVICE_ENV", "0"),
listenAddr:   env("LISTEN_ADDR", ":9000"),
downstream:   configuredDownstream(name),
tenantHeader: env("TENANT_HEADER", "X-Tenant"),
client: &http.Client{
// ���ӳ�ʱ�����临�ӵ��������
Timeout: 10 * time.Second,
},
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

log.Printf("Istio-native Service starting: %s (Env: %s) -> Downstream: %s", s.name, s.env, valueOrDash(s.downstream))

if err := server.ListenAndServe(); err != nil {
log.Fatal(err)
}
}

func (s *service) handle(w http.ResponseWriter, r *http.Request) {
n := r.URL.Query().Get("n")
if n == "" { n = "0" }

// ���������ģ�����Ⱦɫ (Traffic Staining)
// �� URL ������ȡ�⻧ ID ��ע�뵽 Header���� Istio Sidecar ʶ��� Header ����·�ɡ�
tenant := s.tenantFrom(r)

if s.downstream == "" {
w.Header().Set("X-Service-Name", s.name)
w.Header().Set("X-Service-Env", s.env)
fmt.Fprintf(w, "Endpoint: service=%s env=%s n=%s tenant=%s\n", s.name, s.env, n, tenant)
return
}

// ���Եķ�������ֱ�ӵ��� K8s ��������
// Sidecar ���������󣬽��� X-Tenant Header�������� VirtualService ����ִ�С���Ӿ����·�ɡ�
targetURL := fmt.Sprintf("http://%s:9000/", s.downstream)

resp, err := s.callDownstream(targetURL, n, tenant, r.Header)
if err != nil {
http.Error(w, fmt.Sprintf("Mesh routing error: %v", err), http.StatusBadGateway)
return
}
defer resp.Body.Close()

// ��Ӧ͸��
for k, vv := range resp.Header {
for _, v := range vv { w.Header().Add(k, v) }
}
w.WriteHeader(resp.StatusCode)
io.Copy(w, resp.Body)
}

func (s *service) tenantFrom(r *http.Request) string {
// ���ȴ� Header ȡ��͸�������ģ���û����� Query ȡ����ڴ������ģ�
if t := r.Header.Get(s.tenantHeader); t != "" { return t }
return r.URL.Query().Get("tenant")
}

func (s *service) callDownstream(targetURL, n, tenant string, originalHeaders http.Header) (*http.Response, error) {
u, _ := url.Parse(targetURL)
q := u.Query()
q.Set("n", n)
if tenant != "" { q.Set("tenant", tenant) }
u.RawQuery = q.Encode()

req, _ := http.NewRequest(http.MethodGet, u.String(), nil)

// ����һ/�����еĹؼ���ȫ��·͸��
// ����͸������ X- ͷ���������⻧������� Istio ׷�ٱ꣨Trace IDs��
for k, vv := range originalHeaders {
lk := strings.ToLower(k)
if strings.HasPrefix(lk, "x-") || strings.HasPrefix(lk, "grpc-") {
for _, v := range vv { req.Header.Add(k, v) }
}
}

// ȷ����ǰ��ҵ��Ⱦɫ�걻��ʽ���͵����Σ����� Envoy ·��
if tenant != "" {
req.Header.Set(s.tenantHeader, tenant)
}

return s.client.Do(req)
}

func env(key, fallback string) string {
if v := os.Getenv(key); v != "" { return v }
return fallback
}

func valueOrDash(s string) string {
if s == "" { return "-" }
return s
}

func configuredDownstream(name string) string {
var id int
if _, err := fmt.Sscanf(name, "svc%d", &id); err != nil { return "" }
if id >= 15 { return "" }
return fmt.Sprintf("svc%d", id+1)
}

