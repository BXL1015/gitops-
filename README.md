# 微服务全链路泳道实验 (Istio 版)

本项目已从自定义治理方案升级为 **Istio 服务网格** 方案，用于学习和验证云原生流量管控。

## 1. 实验架构
- **基础设施**：Istio 控制平面负责流量调度。
- **业务代码**：[internal/business/service.go](internal/business/service.go) 已简化，仅保留 **Header 透传** 逻辑。
- **调用链**：svc1 -> svc2 -> svc3 (路由点) -> ... -> svc15。

## 2. 核心治理规则 (泳道)
实验模拟了开发联调场景：
- **基线环境 (env0)**：部署全量服务。
- **测试泳道 (env8)**：仅部署部分服务。
- **路由逻辑**：
  - 当请求头 X-Tenant: alice 时，svc3 会自动路由到泳道版本。
  - Istio 会处理后续的 Fallback（如果泳道内没有下游服务，自动回到基线）。

## 3. 极速上手 (k3d + Istio)

### 第一步：安装 Istio
`powershell
# 如果你还没安装 istioctl
curl -L https://istio.io/downloadIstio | sh -
# 安装至集群并开启自动注入
istioctl install --set profile=demo -y
kubectl label namespace default istio-injection=enabled
`

### 第二步：部署服务
由于内存受限，建议仅部署核心链路进行测试：
`powershell
# 部署基线环境 (建议按需部署 svc1~svc5)
kubectl apply -f k8s/env0/

# 部署 Istio 路由规则
kubectl apply -f k8s/istio/routing.yaml
`

### 第三步：验证
`powershell
# 普通请求 (流经 env0)
curl "http://<svc1-ip>:9000/?n=10"

# 泳道请求 (将触发 env8 路由)
curl -H "X-Tenant: alice" "http://<svc1-ip>:9000/?n=10"
`

---
*注：本项目已移除 registry 和 tenant-lookup 服务，全面拥抱云原生。*
