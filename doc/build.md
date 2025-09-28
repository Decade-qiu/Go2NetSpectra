# Go2NetSpectra 构建与开发指南

本文档旨在为开发者提供一个清晰的指南，说明如何配置开发环境、构建和运行 Go2NetSpectra 项目的各个组件。

---

## 1. 环境与配置

### 1.1. 环境准备 (Prerequisites)

在开始之前，请确保您的开发环境中已安装以下工具：

- **Go**: 版本 `1.21` 或更高。请通过 `go version` 命令确认。
- **Protobuf Compiler (`protoc`)**: 用于将 `.proto` 文件编译成 Go 代码。请从 [Protobuf GitHub Releases](https://github.com/protocolbuffers/protobuf/releases) 下载并安装。
- **Docker**: 用于快速启动 NATS、ClickHouse 等依赖服务，以及进行容器化部署。
- **`godotenv`**: Go 应用程序用于加载 `.env` 文件的库，通过 `go mod` 自动管理。

### 1.2. 配置文件说明

Go2NetSpectra 采用 **`configs/config.yaml`** 作为唯一的配置文件。为了实现灵活的环境配置和敏感数据管理，我们利用了 **环境变量**。

- **`configs/config.yaml`**: 包含所有应用程序的配置项。敏感数据和环境相关的设置（如服务地址、端口、凭证）都使用 `${VAR_NAME}` 占位符。程序启动时会读取此文件，并通过 `os.ExpandEnv` 自动替换这些占位符。

- **`.env` 文件 (本地开发)**: 在项目根目录下创建 `.env` 文件（可从 `configs/.env.example` 复制）。此文件用于存储本地开发环境的特定配置（例如 `NATS_URL=nats://localhost:4222`）和敏感凭证。Go 应用程序在启动时会自动加载此文件。

- **`.docker.env` 文件 (Docker Compose)**: 在 `deployments/docker-compose/` 目录下创建 `.docker.env` 文件（可从 `configs/.env.example` 复制）。此文件用于存储 Docker Compose 环境的特定配置（例如 `NATS_URL=nats://nats:4222`）和敏感凭证。`docker-compose` 会自动加载此文件，并将变量传递给容器。

**重要提示**: `.env` 和 `.docker.env` 文件都已被添加到 `.gitignore` 中，请勿将其提交到版本控制系统。

---

## 2. 生成 Protobuf 代码

项目使用 Protobuf 来定义跨服务传输的数据结构。在初次克隆项目或修改了 `api/proto/v1/` 目录下的 `.proto` 文件后，您需要重新生成 Go 代码。

**第一步：安装 Go 插件**

如果您尚未安装 `protoc` 的 Go 语言插件，请运行以下命令：
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

**第二步：生成代码**

在项目根目录下，执行以下命令来生成所有 `.proto` 文件：
```sh
protoc --proto_path=api/proto \
       --go_out=. --go-grpc_out=. \
       api/proto/v1/*.proto
```

命令成功后，会在 `api/gen/v1/` 目录下生成或更新对应的 `.pb.go` 和 `_grpc.pb.go` 文件。

---

## 3. 本地开发模式

在本地开发模式下，我们通常在本地运行 Go 程序，但将 NATS 和 ClickHouse 等依赖作为 Docker 容器启动。

### 3.1. 环境准备

1.  **创建 `.env` 文件**: 在项目根目录下，复制 `configs/.env.example` 到 `.env`，并根据您的本地环境填写所有变量。
2.  **启动依赖服务**: 在不同终端中启动 NATS 和 ClickHouse 容器。

```sh
# 终端 1: 启动 NATS
docker run --rm -p 4222:4222 nats:latest

# 终端 2: 启动 ClickHouse (注意端口映射)
docker run -d -p 18123:8123 -p 19000:9000 -e CLICKHOUSE_PASSWORD=${CLICKHOUSE_PASSWORD} --name some-clickhouse-server --ulimit nofile=262144:262144 clickhouse/clickhouse-server
```

### 3.2. 运行应用

在不同终端中启动核心服务。应用程序将自动从 `.env` 文件中加载配置。

```sh
# 终端 3: 启动引擎
go run ./cmd/ns-engine/main.go

# 终端 4: 启动 API 服务 (v2 gRPC Server)
go run ./cmd/ns-api/v2/main.go

# 终端 5: 启动 AI 服务
go run ./cmd/ns-ai/main.go

# 终端 6: 启动探针
sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
```

---

## 4. 容器化部署模式 (Docker Compose)

这是推荐的、用于快速启动整个后端系统的方式。

### 4.1. 配置

1.  **创建 `.docker.env` 文件**: 在 `deployments/docker-compose/` 目录下，复制 `configs/.env.example` 到 `.docker.env`，并根据您的 Docker Compose 环境填写所有变量。
2.  **确保 `config.yaml` 正确**: `docker-compose` 会将 `configs/config.yaml` 挂载到容器内部，并通过 `.docker.env` 提供的环境变量进行配置。

### 4.2. 启动服务

在 `deployments/docker-compose/` 目录下，运行：
```sh
docker compose up --build
```

此命令会一并启动 `nats`, `clickhouse`, `ns-engine`, `ns-api`, `ns-ai` 和 `grafana` 等所有服务，并处理好它们之间的启动依赖顺序。

### 4.3. 验证与查询

在 `docker compose` 运行后，您可以通过以下方式与系统交互：

*   **访问 Grafana**: 在浏览器中打开 `http://localhost:3000` (默认用户名/密码: `admin`/`admin`)。您应该能看到一个名为 `Go2NetSpectra Overview` 的预置仪表盘，并能从中看到实时数据。

*   **运行探针**: 在一个新终端中，运行 `ns-probe` 向容器化的 NATS 发送数据。确保您的本地 `.env` (或环境变量) 中 `NATS_URL` 配置为 `nats://localhost:4222`。
    ```sh
    sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
    ```

*   **使用查询脚本**: 在另一个新终端中，使用 **v2 脚本** 与 `ns-api` 的 gRPC 服务交互。确保您的本地 `.env` (或环境变量) 中 `API_GRPC_LISTEN_ADDR` 配置为 `localhost:50051`。
    ```sh
    # 查询聚合流
    go run ./scripts/query/v2/main.go --mode=aggregate --task=per_src_ip

    # 查询大流 (heavy hitters)
    go run ./scripts/query/v2/main.go --mode=heavyhitters --task=per_src_ip --type=0 --limit=10

    # 与 AI 服务交互
    go run ./scripts/ask-ai/main.go "Summarize the network traffic anomalies."
    ```

---

## 6. 算法验证与性能测试

项目包含对 `sketch` 等估计算法的单元测试和性能基准测试。

**运行 Count-Min Sketch 准确性与并发测试**:
```sh
go test -v ./internal/engine/impl/sketch/
```

**运行 `sketch` 与 `exact` 的性能对比基准测试**:
```sh
go test -bench=. ./internal/engine/impl/benchmark/
```

---

## 7. 辅助工具

### 5.1. 查询脚本

项目在 `scripts/query/v2/` 目录下提供了一个多功能 gRPC 查询工具，支持多种模式和参数。详情请运行 `go run ./scripts/query/v2/main.go --help`。

### 5.2. Gob 解码器

用于解码 `gob` writer 生成的 `.dat` 文件。

**使用方法**:
```sh
go run ./scripts/gobana/main.go <path_to_dat_file>
```

### 5.3. NATS 消息验证

`ns-probe` 工具内置了订阅者模式，用于快速验证 NATS 主题上是否有数据流过。

**使用方法**:
```sh
go run ./cmd/ns-probe/main.go --mode=sub
```

---

## 5. Kubernetes 部署模式 (高级)

为了在生产或准生产环境中实现高可用和可伸缩的部署，项目提供了完整的 Kubernetes 部署方案。

### 5.1. 环境准备

- 一个正在运行的 Kubernetes 集群。
- `kubectl` 命令行工具，并已配置好与集群的连接。
- `helm` 命令行工具 (如果使用 Helm 部署)。

### 5.2. 方法 A: 使用部署脚本 (适用于快速测试)

此方法通过一个 shell 脚本，按正确的依赖顺序应用 `deployments/kubernetes/` 目录下的所有 YAML 清单文件。

**第一步：配置 Secret**

在部署之前，您必须编辑 `deployments/kubernetes/go2netspectra-secret.yaml` 文件，填入您自己的真实凭证，例如 `AI_API_KEY`, `SMTP` 相关字段以及 `CLICKHOUSE_PASSWORD`。

**第二步：运行脚本**

```sh
# 进入 k8s 部署目录
cd deployments/kubernetes/

# 赋予脚本执行权限
chmod +x deploy-k8s.sh

# 执行一键部署
./deploy-k8s.sh
```

脚本会自动创建所有资源，并等待 NATS 和 ClickHouse 集群进入就绪状态后，再部署应用服务，最后等待所有应用部署完成。

### 5.3. 方法 B: 使用 Helm Chart (推荐)

Helm 是 Kubernetes 的包管理器，使用 Helm 是进行版本化、可配置化部署的最佳实践。

**第一步：自定义配置**

`go2netspectra` Chart 的所有配置项都在 `deployments/helm/go2netspectra/values.yaml` 文件中。建议复制一份进行修改，以免影响原始文件。

```sh
cd deployments/helm/go2netspectra/
cp values.yaml my-custom-values.yaml
```

然后，编辑 `my-custom-values.yaml`，至少需要修改 `config` 部分的 `ai.api_key`, `smtp` 和 `clickhouse` 的密码等敏感信息。

**第二步：安装 Chart**

使用 `helm install` 命令进行安装。`go2netspectra` 是您为这个部署实例起的名字（Release Name）。

```sh
# （可选）先用 --dry-run 模式检查将要生成的 K8s 资源是否正确
helm install go2netspectra . -f my-custom-values.yaml --dry-run --debug

# 如果检查无误，正式安装
helm install go2netspectra . -f my-custom-values.yaml
```

**第三步：检查与卸载**

```sh
# 查看部署状态
helm status go2netspectra

# 卸载部署
helm uninstall go2netspectra
```
