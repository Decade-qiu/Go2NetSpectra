# Go2NetSpectra 构建与开发指南

本文档旨在为开发者提供一个清晰的指南，说明如何配置开发环境、构建和运行 Go2NetSpectra 项目的各个组件。

---

## 1. 环境准备 (Prerequisites)

在开始之前，请确保您的开发环境中已安装以下工具：

- **Go**: 版本 `1.21` 或更高。请通过 `go version` 命令确认。
- **Protobuf Compiler (`protoc`)**: 用于将 `.proto` 文件编译成 Go 代码。请从 [Protobuf GitHub Releases](https://github.com/protocolbuffers/protobuf/releases) 下载并安装。
- **Docker**: 用于快速启动 NATS、ClickHouse 等依赖服务。请确保 Docker 服务正在运行。

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

## 3. 运行离线分析 (`pcap-analyzer`)

`pcap-analyzer` 是一个用于对 `pcap` 文件进行深度分析和聚合的工具。它可以验证核心聚合引擎的逻辑。

**运行命令**:
```sh
go run ./cmd/pcap-analyzer/main.go <pcap_file_path>
```

**示例**:
```sh
go run ./cmd/pcap-analyzer/main.go ./test/data/10M.pcap
```

聚合结果的快照将保存在 `configs/config.yaml` 中 `aggregator.exact.writers` 下 `type: gob` 的 `writer` 的 `gob.root_path` 定义的路径下。请注意，`aggregator.period` 定义了全局的测量周期，而每个 `writer` 的 `snapshot_interval` 则控制了其独立的快照频率。

---

## 4. 运行实时数据采集与处理

这是项目的核心实时流水线，由 `ns-probe` 采集数据，`ns-engine` 处理数据。

**操作流程**:

您需要打开 **三个独立** 的终端窗口来运行整套流水线。

**终端 1: 启动 NATS 服务器**

使用 Docker 启动一个 NATS 服务器实例：
```sh
docker run --rm -p 4222:4222 -ti nats:latest
```

**终端 2: 启动实时聚合引擎 (ns-engine)**

`ns-engine` 会连接到 NATS，订阅数据并启动内部的并发处理引擎。它的行为由 `configs/config.yaml` 定义，特别是 `aggregator.period` (全局测量周期) 和 `aggregator.exact.writers` (可独立启用和配置快照间隔的写入器)。
```sh
go run ./cmd/ns-engine/main.go
```

**终端 3: 启动探针 (ns-probe)**

探针将从指定的网络接口抓包，并发布到 NATS。请将 `<interface_name>` 替换为您的实际网卡名（如 `en0`, `eth0`）。

**注意**: 抓包通常需要管理员权限。
```sh
sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
```

操作正确的话，您将在**终端 2** (`ns-engine`) 中看到 `Manager started with...`、周期性的 `Started snapshotter for a writer with interval...` 和 `Started global resetter with period...` 日志。

---

## 5. 运行 API 服务与查询

`ns-api` 服务提供 RESTful API 用于查询 ClickHouse 中的聚合数据。

### 5.1. 启动依赖服务 (ClickHouse)

使用 Docker 启动一个 ClickHouse 服务器实例。请注意，我们将主机的 `19000` 端口映射到容器的 `9000` 端口，并设置了密码。

```sh
docker run -d -p 18123:8123 -p 19000:9000 -e CLICKHOUSE_PASSWORD=123 --name some-clickhouse-server --ulimit nofile=262144:262144 clickhouse/clickhouse-server
```

### 5.2. 配置 `ns-engine`

确保您的 `configs/config.yaml` 文件中，`clickhouse` writer 已被启用并配置了正确的密码：

```yaml
# ...
      - type: "clickhouse"
        enabled: true
        snapshot_interval: "60s"
        clickhouse:
          host: "localhost"
          port: 19000  # <-- 确保端口与 docker run 命令中的映射一致
          database: "default"
          username: "default"
          password: "123" # <-- 确保密码与 docker run 命令中的设置一致
# ...
```

### 5.3. 启动 `ns-api` 服务

在项目根目录下运行：
```sh
go run ./cmd/ns-api/main.go
```
您应该会看到日志 `API server starting on :8080`。

### 5.4. 发送查询请求

项目在 `scripts/query/` 目录下提供了一个多功能查询工具。

*   **通过 API 查询 (默认模式)**:
    ```sh
    go run ./scripts/query/main.go
    ```

*   **通过 API 查询特定任务**: 
    ```sh
    go run ./scripts/query/main.go -mode=api -task=per_src_ip
    ```

*   **直接查询 ClickHouse (用于验证)**:
    ```sh
    go run ./scripts/query/main.go -mode=direct -task=per_five_tuple
    ```

---

## 6. 辅助工具

### 6.1. Gob 解码器

项目提供了一个脚本，用于解码和查看由 `pcap-analyzer` 或 `ns-engine` 生成的 `.dat` 快照文件的内容。这些文件由 `gob` writer 生成。

**使用方法**:
```sh
go run ./scripts/gobana/main.go <path_to_dat_file>
```

### 6.2. NATS 消息验证

`ns-probe` 工具内置了订阅者模式，可以用来快速验证 NATS 主题上是否有数据流过。

**使用方法**:
```sh
go run ./cmd/ns-probe/main.go --mode=sub
```

### 6.3. 探针持久化

`ns-probe` 支持将捕获到的数据包异步写入本地文件，以便后续分析或备份。此功能通过 `configs/config.yaml` 中的 `probe.persistence` 部分进行配置。

- **`enabled`**: `true` 或 `false`，控制是否开启持久化。
- **`path`**: 持久化文件存储的**目录**。
- **`encoding`**: 存储格式，支持 `text`（人类可读的五元组信息）、`gob`（Go特定的二进制格式）和 `pcap`（标准网络抓包格式）。
- **`num_workers`**: 用于写入文件的协程数量。
- **`channel_buffer_size`**: 内存中用于缓冲数据包的通道大小。

当 `enabled: true` 时，`ns-probe` 会在指定的 `path` 目录下创建一个以启动时间戳命名的文件（如 `2025-09-13_10-30-00.pcap`），并将数据写入其中。