# Baseline Kubernetes Manifests

This directory is the baseline `env=0` deployment set.

Business service manifests:

```text
svc1.yaml ... svc15.yaml
```

Infrastructure manifests:

```text
registry.yaml
tenant-lookup.yaml
```

Each business manifest contains one `Deployment` and one `Service`. The Pod
reads its identity from environment variables:

```text
SERVICE_NAME=svcN
SERVICE_ENV=0
PUBLIC_ADDR=http://svcN:9000
REGISTRY_ADDR=http://registry:8080
TENANT_LOOKUP_ADDR=http://tenant-lookup:8081
```

For a SET environment, the GitOps pipeline should copy or generate only the
required hot service manifests, change names/selectors/service names to include
the env id, and set `SERVICE_ENV` plus `PUBLIC_ADDR` to the target env.
