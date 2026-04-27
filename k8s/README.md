# K8S Manifest Repository

This directory is organized by environment. Argo CD should point to one environment directory at a time.

- `env0/`: baseline environment manifests
- `envN/`: generated SET environment manifests, for example `env2/`

The baseline environment is not named `base`; it is environment `0`.