# Go2NetSpectra 构建与开发指南

本文档旨在为开发者提供一个清晰的指南，说明如何配置开发环境、构建和运行 Go2NetSpectra 项目的各个组件。

---

## 1. 环境与配置

### 1.1. 环境准备 (Prerequisites)

在开始之前，请确保您的开发环境中已安装以下工具：

- **Go**: 版本 `1.21` 或更高。请通过 `go version` 命令确认。
- **Protobuf Compiler (`protoc`)**: 用于将 `.proto` 文件编译成 Go 代码。请从 [Protobuf GitHub Releases](https://github.com/protocolbuffers/protobuf/releases) 下载并安装。
- **Docker**: 用于快速启动 NATS、ClickHouse 等依赖服务，以及进行容器化部署。

### 1.2. 配置文件说明

项目使用两个独立的配置文件来管理不同环境下的设置：

- **`configs/config.yaml`**: 用于**本地开发**。当您在本地使用 `go run` 命令直接运行 `ns-probe`, `ns-engine`, `ns-api` 时，程序会加载此文件。在此文件中，所有服务地址（如 NATS, ClickHouse）都应配置为 `localhost`，以便连接到通过 Docker 暴露在主机上的端口。

- **`configs/config.docker.yaml`**: 用于**容器化部署**。当您使用 `docker compose` 启动服务时，此文件会被构建到镜像内部。在此文件中，所有服务地址都必须使用 Docker Compose 服务名（如 `nats`, `clickhouse`），以便容器在内部网络中互相发现。

---

## 2. 生成 Protobuf 代码

项目使用 Protobuf 来定义跨服务传输的数据结构。在初次克隆项目或修改了 `api/proto/v1/` 目录下的 `.proto` 文件后，您需要重新生成 Go 代码。

**第一步：安装 Go 插件**

如果您尚未安装 `protoc` 的 Go 语言插件，请运行以下命令：
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
```

**第二步：生成代码**

在项目根目录下，执行以下命令来生成所有 `.proto` 文件：
```sh
protoc --proto_path=api/proto --go_out=. api/proto/v1/*.proto
```

命令成功后，会在 `api/gen/v1/` 目录下生成或更新对应的 `.pb.go` 文件。

---

## 3. 本地开发模式

在本地开发模式下，我们通常在本地运行 Go 程序，但将 NATS 和 ClickHouse 等依赖作为 Docker 容器启动。

### 3.1. 启动依赖服务

```sh
# 终端 1: 启动 NATS
docker run --rm -p 4222:4222 nats:latest

# 终端 2: 启动 ClickHouse (注意端口映射)
docker run -d -p 18123:8123 -p 19000:9000 -e CLICKHOUSE_PASSWORD=123 --name some-clickhouse-server --ulimit nofile=262144:262144 clickhouse/clickhouse-server
```

### 3.2. 运行应用

确保您的 `configs/config.yaml` 文件中的地址指向 `localhost`（例如 `nats_url: nats://localhost:4222`, `host: localhost`, `port: 19000`）。

```sh
# 终端 3: 启动引擎
go run ./cmd/ns-engine/main.go

# 终端 4: 启动 API 服务
go run ./cmd/ns-api/main.go

# 终端 5: 启动探针
sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
```

---

## 4. 容器化部署模式 (Docker Compose)

这是推荐的、用于快速启动整个后端系统的方式。

### 4.1. 配置

`docker compose` 模式会自动使用 `configs/config.docker.yaml` 文件，该文件已被配置为使用服务名进行容器间通信。您无需修改它。

### 4.2. 启动服务

在 `deployments/docker-compose/` 目录下，运行：
```sh
docker compose up --build
```

此命令会一并启动 `nats`, `clickhouse`, `ns-engine`, 和 `ns-api` 四个服务，并处理好它们之间的启动依赖顺序。

### 4.3. 验证与查询

在 `docker compose` 运行后，您可以通过以下方式与系统交互：

*   **访问 Grafana**: 在浏览器中打开 `http://localhost:3000` (默认用户名/密码: `admin`/`admin`)。您应该能看到一个名为 `Go2NetSpectra Overview` 的预置仪表盘，并能从中看到实时数据。

*   **运行探针**: 在一个新终端中，运行 `ns-probe` 向容器化的 NATS 发送数据。
    ```sh
    sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
    ```

*   **使用查询脚本**: 在另一个新终端中，使用脚本与 `ns-api` 交互。
    ```sh
    go run ./scripts/query/main.go -mode=aggregate
    ```

---

## 6. 算法验证测试

项目包含对 `sketch` 等估计算法的单元测试，用于验证其准确性。

**运行 Count-Min Sketch 测试**:
```sh
go test -v ./internal/engine/impl/sketch/
```

---

## 7. 辅助工具

### 5.1. 查询脚本

项目在 `scripts/query/` 目录下提供了一个多功能查询工具，支持多种模式和参数。详情请运行 `go run ./scripts/query/main.go --help`。

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
