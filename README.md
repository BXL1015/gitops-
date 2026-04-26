# GitOps Lane Simulation

This repo is a minimal Go-only simulation for multi-SET / lane-style
microservice routing.

## Programs

- `registry`: service registry, stores `service name + env -> address`.
- `tenant-lookup`: simulates the SEaaS / tenant platform, stores
  `tenant/account -> env`.
- `svc1` ... `svc15`: business services.

All programs use only the Go standard library.

## CI Image

The GitHub Actions workflow at `.github/workflows/image-ci.yml` builds and
pushes one image per program to GitHub Container Registry:

```text
ghcr.io/bxl1015/gitops-lane-svc3:<tag>
ghcr.io/bxl1015/gitops-lane-svc4:<tag>
```

In Kubernetes, each service manifest points to its own image. The Pod still
reads service identity and environment from YAML environment variables:

```yaml
env:
  - name: SERVICE_NAME
    value: svc3
  - name: SERVICE_ENV
    value: "4"
  - name: REGISTRY_ADDR
    value: http://registry:8080
  - name: PUBLIC_ADDR
    value: http://svc3-env4:9000
```

The workflow runs on pushes to `master` and `dev`. It can also be started
manually with `workflow_dispatch`; the optional `set_env` input only affects the
image tag, for example `set-4-abcdef1`. CD and Argo CD are intentionally left
out for the next step.

## Default Topology

If `DOWNSTREAM_SERVICE` is not set, the business services form this chain:

```text
svc1 -> svc2 -> svc3 -> svc4 -> svc5 -> svc6 -> svc7 -> svc8 -> svc9 -> ... -> svc15
```

Set `DOWNSTREAM_SERVICE=-` or `DOWNSTREAM_SERVICE=none` to make a service a
terminal node.

## Service Layers

The demo assumes:

```text
hot SET services: svc3 ~ svc8
baseline/cold services: svc1, svc2, svc9 ~ svc15
default boundary service: svc2
```

`svc2` is a boundary because its downstream `svc3` is the beginning of the hot
SET layer. It can query `tenant-lookup` with the request's business tenant id
and use the returned env for the next hop.

After the request falls back to a baseline/cold service, later calls continue
from that service's own `SERVICE_ENV=0`, unless another boundary service is
explicitly enabled.

## Routing Modes

Each business service supports:

```text
LOOKUP_MODE=none      # default for most services
LOOKUP_MODE=boundary  # default for svc2
LOOKUP_MODE=always    # useful to simulate the expensive "every hop lookup" plan
```

`boundary` and `always` currently behave the same in code; the names exist so
the experiment can describe intent clearly.

## Request Identifier

The simulation forwards a business identifier named `tenant`:

```text
curl "http://127.0.0.1:9001/?tenant=alice&n=100"
```

This is not an env marker. It represents a real production identifier such as
an account id, employee id, user id, or tenant id. Boundary services use it to
query `tenant-lookup`.

## Example

Start registry:

```bash
go run ./registry
```

Start tenant lookup:

```bash
TENANT_BINDINGS=alice=1,bob=2 go run ./tenant-lookup
```

Start baseline `svc1`:

```bash
SERVICE_ENV=0 LISTEN_ADDR=:9001 PUBLIC_ADDR=http://127.0.0.1:9001 go run ./svc1
```

Start hot `svc3` in env 1:

```bash
SERVICE_ENV=1 LISTEN_ADDR=:9103 PUBLIC_ADDR=http://127.0.0.1:9103 go run ./svc3
```

Call the baseline entrance:

```bash
curl "http://127.0.0.1:9001/?tenant=alice&n=100"
```

Expected route shape:

```text
svc1 env0 -> svc2 env0 -> tenant lookup says alice belongs to env1
svc2 routes to svc3 env1
svc3~svc8 stay in env1
svc8 cannot find svc9 env1, falls back to svc9 env0
svc9~svc15 stay in env0
```
