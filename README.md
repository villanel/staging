# 更安全的PR测试：共享 staging 集群原型方案

## 任务概述

### 问题背景
当前，当我们直接将PR部署到共享的Kubernetes staging环境时，可能会破坏其他也在使用staging环境的人的工作。我们需要一个更好的方法来测试PR，同时确保staging环境的稳定性。

### 假设条件
- 已有的共享Kubernetes staging环境
- 单monorepo中的多个服务/组件
- 希望改进PR合并前的测试方式

### 目标
设计并实现一个原型解决方案，实现：
- 为每个PR，将该PR版本的服务部署到staging集群
- 可以针对PR版本运行测试
- 正常的staging环境继续为其他人工作（PR部署不应破坏共享staging）

### 技术选型
- **Istio**：用于流量管理、服务版本控制和基于唯一标头的请求路由，实现多个环境在共享集群中共存
- **OpenTelemetry**：不仅用于可观测性收集和分析，更重要的是实现请求级租户上下文注入和跨服务边界的元数据传播，无需完全隔离的基础设施

## 解决方案设计

### 核心设计理念
1. **服务版本隔离**：为每个PR部署独立版本的服务，使用唯一标识（如PR编号）进行区分
2. **请求级租户管理**：通过Istio和OpenTelemetry实现高效的请求级租户管理，无需完全隔离的基础设施
3. **智能流量路由**：使用Istio基于唯一标头路由和分割每个环境的请求，允许多个环境共存，同时最大限度地减少资源消耗并保持逻辑隔离
4. **上下文传播**：利用OpenTelemetry的baggage传播机制，自动在服务之间传递特定于环境的元数据，实现一致的环境特定行为和无缝重新路由
5. **完整可观测性**：使用OpenTelemetry收集PR版本服务的指标、痕迹和日志，便于调试和监控
6. **CI/CD自动化**：将PR部署、测试和清理流程集成到CI管道中

### 架构图
```
┌───────────────────────────────────────────────────────────────────┐
│                           CI/CD Pipeline                          │
└───────────┬───────────────────────────────────────────────────────┘
            │
            ▼
┌───────────────────────────────────────────────────────────────────┐
│                     Kubernetes Staging Cluster                    │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                          Istio Mesh                         │  │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │  │
│  │  │   Service   │  │   Service   │  │   Service   │          │  │
│  │  │  (PR #123)  │  │  (PR #456)  │  │  (Stable)   │          │  │
│  │  └─────────────┘  └─────────────┘  └─────────────┘          │  │
│  │          ▲                ▲                ▲                │  │
│  │          │                │                │                │  │
│  │          └────────┬───────┴───────┬────────┘                │  │
│  │                   │               │                         │  │
│  │  ┌─────────────────────────────────────────────────────┐    │  │
│  │  │                   Istio Ingress Gateway             │    │  │
│  │  └─────────────────────────────────────────────────────┘    │  │
│  │                          ▲                                   │  │
│  │       ┌──────────────────┴──────────────────┐               │  │
│  │       │                                      │               │  │
│  │       ▼                                      ▼               │  │
│  │  ┌─────────────┐                      ┌─────────────┐       │  │
│  │  │   PR流量     │                      │   正常流量   │       │  │
│  │  │  (带x-pr-id头)│                      │  (无x-pr-id头)│       │  │
│  │  └─────────────┘                      └─────────────┘       │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
            │
            ▼
┌───────────────────────────────────────────────────────────────────┐
│                     OpenTelemetry Collector                        │
└───────────┬───────────────────────────────────────────────────────┘
            │
            ▼
┌───────────────────────────────────────────────────────────────────┐
│                     Observability Backend                          │
│  (Jaeger for traces, Prometheus for metrics, Loki for logs)       │
└───────────────────────────────────────────────────────────────────┘
```

**流量路由说明**：
1. **PR流量**：带有`x-pr-id`头的请求 → Istio Ingress Gateway → 根据头值路由到对应PR版本的服务
2. **正常流量**：无`x-pr-id`头的请求 → Istio Ingress Gateway → 路由到Stable版本的服务

### 关键组件说明

#### 1. Istio 服务网格
- **VirtualService**：定义流量路由规则，根据请求头或其他属性将流量导向特定PR版本
- **DestinationRule**：定义服务的不同子集（subset），每个PR版本对应一个子集
- **Gateway**：处理外部流量入口，支持基于主机名或路径的路由

#### 2. OpenTelemetry 可观测性与上下文传播
- **自动 instrumentation**：为服务添加分布式追踪支持
- **上下文注入**：将PR编号等租户标识注入到请求上下文中
- **Baggage 传播**：利用OpenTelemetry的baggage机制，自动在服务之间传递特定于环境的元数据
- **PR标签注入**：将PR编号作为属性注入到所有遥测数据中
- **统一收集**：通过OpenTelemetry Collector收集所有服务的遥测数据
- **关联分析**：在观测平台中按PR编号过滤和关联遥测数据
- **请求级租户管理**：实现高效的请求级租户管理，无需完全隔离的基础设施

## 实现步骤

### 1. 环境准备
- 确保Kubernetes集群已安装Istio
- 安装OpenTelemetry Collector

### 2. 服务改造
#### 2.1 OpenTelemetry Baggage注入实现
我们的代码库已经在`pkg/otel`目录中实现了完整的OpenTelemetry Baggage注入机制：

##### 核心实现文件
- **`pkg/otel/init.go`**：初始化OpenTelemetry Tracer，配置包含Baggage的传播器
- **`pkg/otel/hertz_middleware.go`**：Hertz框架中间件，实现从请求头提取PR标识并注入到Baggage中

##### Baggage注入工作原理
1. **初始化配置**：在`init.go`中配置传播器，确保包含Baggage：
   ```go
   otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
       propagation.TraceContext{},
       propagation.Baggage{},
   ))
   ```

2. **中间件提取与注入**：在`hertz_middleware.go`的Middleware函数中：
   - 从请求头`x-pr-id`中提取PR标识
   - 创建baggage member并添加到上下文中
   - 自动传播到后续服务调用

3. **服务使用**：各个服务（productpage、ratings、reviews、details）通过导入`pkg/otel`包并使用中间件来实现Baggage注入

#### 2.2 服务集成步骤
1. **导入otel包**：在服务的main.go中导入`pkg/otel`
2. **初始化Tracer**：调用`otel.InitTracer`初始化OpenTelemetry Tracer
3. **使用中间件**：为Hertz服务器添加`otel.Middleware`中间件
4. **配置环境变量**：设置`PR_ID`环境变量，为服务添加PR标识

#### 2.3 代码示例
以productpage服务为例：
```go
package main

import (
    "log"
    "os"

    "github.com/cloudwego/hertz/pkg/app/server"
    "github.com/cloudwego/hertz/pkg/common/hlog"
    "your-repo/pkg/otel"
)

func main() {
    // 1. 初始化OpenTelemetry Tracer
    shutdown, err := otel.InitTracer("productpage")
    if err != nil {
        log.Fatalf("Failed to initialize tracer: %v", err)
    }
    defer shutdown(nil)

    // 2. 创建Hertz服务器
    h := server.Default()

    // 3. 添加OpenTelemetry中间件
    h.Use(otel.Middleware("productpage"))

    // 4. 定义路由和处理函数
    // ...

    // 5. 启动服务器
    hlog.Infof("Starting productpage server...")
    h.Spin()
}
```

#### 2.4 配置示例
为服务注入PR标识环境变量：
```yaml
# Kubernetes Deployment配置示例
spec:
  containers:
  - name: productpage
    image: my-registry/productpage:pr-123
    env:
    - name: PR_ID
      value: "123"
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: "otel-collector:4317"
    - name: OTEL_TRACES_SAMPLER
      value: "always_on"
```

### 3. Istio 配置示例

#### 3.1 HTTPRoute配置示例
实际的HTTPRoute配置通过`k8s/pr-env.yaml.tmpl`模板生成，示例如下：

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: ${SERVICE_NAME}-pr-${PR_ID}-route
  namespace: bookinfo
spec:
  parentRefs:
  - group: ""
    kind: Service
    name: ${SERVICE_NAME} # 拦截发送给原始 Service 的流量
    port: ${SERVICE_PORT}
  rules:
  # --- 规则 A: 匹配 PR 染色流量 (高优先级) ---
  - matches:
    - headers:
      - type: RegularExpression
        name: "baggage"
        value: ".*pr_id=${PR_ID}.*"
    backendRefs:
    - name: ${SERVICE_NAME}-pr-${PR_ID} # 转发给 PR 专用的 Service
      port: ${SERVICE_PORT}

  # --- 规则 B: 兜底逻辑 (低优先级) ---
  # 不需要任何 matches，代表匹配所有其他流量
  - backendRefs:
    - name: ${SERVICE_NAME}-stable # 转发回原始 Service 本身
      port: ${SERVICE_PORT}
```

**配置说明**：
- HTTPRoute名称包含服务名称和PR_ID，确保唯一性
- 规则A：匹配带有`baggage: .*pr_id=${PR_ID}.*`的请求，转发到PR版本服务
- 规则B：兜底逻辑，将所有其他流量转发到stable版本服务
- CI自动生成并部署此配置，无需手动干预



### 4. CI/CD 集成

#### 4.1 典型CI流程
1. **PR创建/更新**：触发CI管道
2. **路径检测**：识别变动的服务
3. **构建镜像**：
   - 从`github.event.number`获取PR编号
   - 使用`pr-{PR_ID}`标签构建服务镜像
   - 将PR编号作为`PR_ID`环境变量注入到Docker镜像中
4. **生成Kubernetes配置**：
   - 使用`k8s/pr-env.yaml.tmpl`模板生成部署文件
   - 为PR版本的服务创建Deployment，设置`PR_ID`环境变量
   - 创建PR版本专用的Service
   - 创建HTTPRoute，匹配带有`baggage: .*pr_id=${PR_ID}.*`的请求
5. **部署PR版本**：
   - 在staging集群中部署PR版本的服务、Service和HTTPRoute
   - HTTPRoute规则A：匹配带有PR标识的baggage头，转发到PR版本服务
   - HTTPRoute规则B：兜底逻辑，将所有其他流量转发到stable版本服务
6. **运行测试**：
   - 发送带有`x-pr-id: {PR_ID}`头的测试请求
   - 中间件自动将`x-pr-id`头转换为baggage的`pr_id`键
   - 收集测试结果和可观测性数据
7. **生成报告**：将测试结果和可观测性数据关联，生成PR测试报告
8. **清理资源**：PR合并或关闭后，删除PR版本的服务、Service和HTTPRoute

#### 4.2 CI脚本示例（GitHub Actions）
实际的CI配置位于`.github/workflows/ci.yaml`，主要包含以下任务：

1. **路径检测**：识别变动的服务
2. **并行构建与推送**：为变动的服务构建Docker镜像并推送
3. **渲染并发布K8s配置**：生成并发布Kubernetes部署文件

```yaml
name: Bookinfo Precise Multi-Platform CI/CD

on:
  pull_request:
    branches: [ main ]
  push:
    branches: [ main ]
    tags: [ 'v*' ]

jobs:
  # 任务 1: 路径检测 (识别变动服务)
  detect-changes:
    runs-on: ubuntu-latest
    outputs:
      services: ${{ steps.filter.outputs.changes }}
    steps:
      - uses: actions/checkout@v4
      - uses: dorny/paths-filter@v3
        id: filter
        with:
          filters: |
            productpage: ['productpage/**', 'pkg/**', 'go.mod']
            reviews: ['reviews/**', 'pkg/**', 'go.mod']
            ratings: ['ratings/**', 'pkg/**', 'go.mod']
            details: ['details/**', 'pkg/**', 'go.mod']

  # 任务 2: 并行构建与推送 (GHCR)
  build-and-push:
    needs: detect-changes
    if: ${{ needs.detect-changes.outputs.services != '[]' }}
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: ${{ fromJson(needs.detect-changes.outputs.services) }}
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      
      # 省略部分步骤...
      
      - name: Set PR ID
        id: pr-info
        run: |
          if [ "${{ github.event_name }}" == "pull_request" ]; then
            echo "pr_id=${{ github.event.number }}" >> $GITHUB_OUTPUT
          else
            echo "pr_id=none" >> $GITHUB_OUTPUT
          fi
      
      - name: Build and push Docker image
        uses: docker/build-push-action@v5
        with:
          context: .
          file: ${{ matrix.service }}/Dockerfile.${{ matrix.service }}
          push: true
          tags: |
            ${{ env.REGISTRY }}/${{ env.IMAGE_BASE_NAME }}-${{ matrix.service }}:pr-${{ steps.pr-info.outputs.pr_id }}
          build-args: |
            PR_ID=${{ steps.pr-info.outputs.pr_id }}

  # 任务 3: 渲染并发布 K8s 配置 (Manifests)
  generate-and-publish:
    needs: [detect-changes, build-and-push]
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: ${{ fromJson(needs.detect-changes.outputs.services) }}
    steps:
      - uses: actions/checkout@v4
      
      - name: Generate Final YAML via Template
        env:
          PR_ID: ${{ github.event.number || 'latest' }}
          SERVICE_NAME: ${{ matrix.service }}
          # 省略其他环境变量...
        run: |
          mkdir -p k8s/generated
          envsubst < k8s/pr-env.yaml.tmpl > k8s/generated/deploy-${{ matrix.service }}.yaml
      
      - name: Upload K8s Artifacts
        uses: actions/upload-artifact@v4
        with:
          name: k8s-manifests-${{ matrix.service }}
          path: k8s/generated/
```

## 服务调用关系

### 服务概述
本项目包含四个核心服务，它们之间通过HTTP接口进行通信，形成完整的调用链：

| 服务名称 | 端口 | 主要功能 | 依赖服务 |
|---------|------|---------|---------|
| productpage | 9083 | 前端服务，展示产品信息 | reviews, details |
| reviews | 9082 | 提供产品评论数据 | ratings |
| ratings | 9080 | 提供产品评分数据 | 无 |
| details | 9081 | 提供产品详情数据 | 无 |

### 调用关系图
```
┌─────────────────┐
│  productpage    │
└────────┬────────┘
         │
         ├───────────┐
         ▼           ▼
┌─────────────┐ ┌─────────────┐
│   reviews   │ │   details   │
└────────┬────┘ └─────────────┘
         │
         ▼
┌─────────────────┐
│    ratings      │
└─────────────────┘
```

### 详细调用流程
1. **productpage → details**
   - productpage服务调用details服务的`/details`端点获取产品详情
   - 调用方式：HTTP GET请求
   - 端口：9081

2. **productpage → reviews**
   - productpage服务调用reviews服务的`/reviews`端点获取产品评论
   - 调用方式：HTTP GET请求
   - 端口：9082

3. **reviews → ratings**
   - reviews服务调用ratings服务的`/ratings`端点获取产品评分
   - 调用方式：HTTP GET请求
   - 端口：9080

### 完整调用链
```
Client → productpage → details
Client → productpage → reviews → ratings
```

## 可观测性实现

### OpenTelemetry 配置

#### 1. 服务端配置
PR版本的服务通过CI自动注入以下环境变量：

| 环境变量 | 来源 | 用途 |
|---------|------|------|
| `PR_ID` | CI中的`github.event.number` | 标识PR版本，代码会自动将其转换为`service.pr_id`资源属性 |
| `OTEL_SERVICE_NAME` | 服务名称 | 标识服务名称 |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | 配置文件 | OpenTelemetry Collector地址 |
| `OTEL_TRACES_SAMPLER` | 配置文件 | 追踪采样策略 |
| `OTEL_PROPAGATORS` | 代码中默认配置 | 启用Baggage传播 |

**代码中的自动配置**：
- 在`pkg/otel/init.go`中，代码会自动从`PR_ID`环境变量提取PR标识，并添加为`service.pr_id`资源属性
- 在`pkg/otel/hertz_middleware.go`中，中间件会自动从请求头`x-pr-id`提取PR标识，并注入到baggage的`pr_id`键中
- 传播器已配置为包含Baggage，确保元数据跨服务传递

**CI注入机制**：
- CI通过`k8s/pr-env.yaml.tmpl`模板生成部署文件
- 模板中自动注入`PR_ID`环境变量：
  ```yaml
  env:
  - name: PR_ID
    value: "${PR_ID}"
  ```

#### 2. 收集器配置（otel-collector-config.yaml）
```yaml
exporters:
  otlp:
    endpoint: jaeger:4317
    tls:
      insecure: true
  prometheus:
    endpoint: 0.0.0.0:8889
  loki:
    endpoint: loki:3100
    tls:
      insecure: true

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
  # 处理上下文信息，确保PR标识被正确关联到所有遥测数据
  resource:
    attributes:
    - key: pr.id
      from_attribute: baggage.pr.id
      action: insert

  # 确保baggage中的PR标识被添加到所有跨度中
  attributes:
    actions:
    - key: pr.id
      from_attribute: baggage.pr.id
      action: insert

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [attributes, resource, batch]
      exporters: [otlp]
    metrics:
      receivers: [otlp]
      processors: [attributes, resource, batch]
      exporters: [prometheus]
    logs:
      receivers: [otlp]
      processors: [attributes, resource, batch]
      exporters: [loki]
```

### 观测数据使用

1. **分布式追踪**：使用Jaeger按PR编号过滤和查看完整调用链
2. **指标监控**：使用Prometheus监控PR版本服务的性能指标
3. **日志分析**：使用Loki按PR编号检索和分析日志
4. **关联分析**：将测试结果与遥测数据关联，快速定位问题

## 优势与收益

### 技术优势
1. **请求级租户隔离**：通过OpenTelemetry的上下文传播和Istio的智能路由，实现高效的请求级租户管理，无需完全隔离的基础设施
2. **零侵入性**：服务代码无需大幅修改，主要通过配置和代理实现
3. **灵活路由**：使用Istio基于唯一标头路由和分割每个环境的请求，允许多个环境共存
4. **上下文自动传播**：利用OpenTelemetry的baggage机制，自动在服务之间传递特定于环境的元数据
5. **完整可观测性**：PR版本的所有遥测数据都带有PR标识，便于分析和调试
6. **资源高效**：共享集群资源，避免为每个PR创建独立环境，最大限度地减少资源消耗
7. **逻辑隔离**：在共享基础设施上实现逻辑隔离，确保PR测试不会影响正常staging环境

### 团队收益
1. **安全测试**：PR测试不会影响正常staging环境
2. **快速反馈**：CI自动部署和测试，缩短反馈周期
3. **协作友好**：多个PR可以同时测试，互不干扰
4. **易于调试**：完整的可观测性数据，便于定位问题

## 假设与限制

### 假设
1. Kubernetes集群已安装并配置Istio
2. 所有服务都可以添加OpenTelemetry instrumentation
3. CI/CD系统支持动态环境配置
4. 共享集群有足够的资源容纳多个PR版本

### 限制
1. **资源竞争**：大量PR同时测试可能导致资源不足
2. **配置复杂度**：Istio配置需要仔细管理，避免冲突
3. **清理依赖**：需要确保PR关闭后资源被正确清理
4. **测试隔离**：某些跨服务测试可能仍然受到影响

## 未来工作

1. **自动化配置管理**：开发工具自动生成和管理Istio配置
2. **智能资源管理**：根据PR优先级和资源使用情况动态调整资源分配
3. **增强测试隔离**：实现更严格的网络隔离，确保PR测试完全独立
4. **自助服务门户**：提供UI界面，让开发者查看和管理自己的PR部署
5. **集成更多观测工具**：与现有监控和告警系统集成
6. **扩展到多集群**：支持跨多个集群的PR测试

## 总结

本方案使用Istio和OpenTelemetry实现了一个安全、高效的PR测试原型，核心在于**请求级租户管理**和**上下文自动传播**：

1. **请求级租户隔离**：通过OpenTelemetry的上下文注入和Istio的智能路由，实现了高效的请求级租户管理，无需完全隔离的基础设施
2. **上下文自动传播**：利用OpenTelemetry的baggage机制，自动在服务之间传递特定于环境的元数据，确保一致的环境特定行为
3. **智能流量路由**：
   - 使用Istio HTTPRoute配置基于baggage头的路由规则
   - 带有PR标识的请求（如`baggage: pr_id=latest`）被路由到对应PR版本的服务
   - 所有其他请求被路由到stable版本的服务
   - 允许多个PR环境在共享集群中共存，同时最大限度地减少资源消耗并保持逻辑隔离
4. **完整可观测性**：所有遥测数据都带有PR标识，便于分析和调试
5. **资源高效**：共享集群资源，避免为每个PR创建独立环境
6. **自动化集成**：与CI/CD系统无缝集成，实现PR部署、测试和清理的自动化

通过这种方式，我们可以在共享staging集群中安全地测试多个PR，同时保持正常staging环境的稳定性，提高开发效率和代码质量。该方案具有良好的扩展性和灵活性，可以根据团队需求进行调整和增强。