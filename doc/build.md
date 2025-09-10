# Go2NetSpectra 构建与开发指南

本文档旨在为开发者提供一个清晰的指南，说明如何配置开发环境、构建和运行 Go2NetSpectra 项目的各个组件。

---

## 1. 环境准备 (Prerequisites)

在开始之前，请确保您的开发环境中已安装以下工具：

- **Go**: 版本 `1.21` 或更高。请通过 `go version` 命令确认。
- **Protobuf Compiler (`protoc`)**: 用于将 `.proto` 文件编译成 Go 代码。请从 [Protobuf GitHub Releases](https://github.com/protocolbuffers/protobuf/releases) 下载并安装。
- **Docker**: 用于快速启动 NATS 等依赖服务。请确保 Docker 服务正在运行。

---

## 2. 生成 Protobuf 代码

项目使用 Protobuf 来定义跨服务传输的数据结构。在初次克隆项目或修改了 `api/proto/v1/traffic.proto` 文件后，您需要重新生成 Go 代码。

**第一步：安装 Go 插件**

如果您尚未安装 `protoc` 的 Go 语言插件，请运行以下命令：
```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
```

**第二步：生成代码**

在项目根目录下，执行以下命令：
```sh
protoc --proto_path=api/proto --go_out=. api/proto/v1/traffic.proto
```

命令成功后，会在 `api/gen/v1/` 目录下生成或更新 `traffic.pb.go` 文件。

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

聚合结果的快照将保存在 `configs/config.yaml` 中 `storage_root_path` 定义的路径下。

---

## 4. 运行实时流量监控

这是项目的核心实时流水线，由 `ns-probe` 采集数据，`ns-engine` 处理数据。

**操作流程**:

您需要打开 **三个独立** 的终端窗口来运行整套流水线。

**终端 1: 启动 NATS 服务器**

使用 Docker 启动一个 NATS 服务器实例：
```sh
docker run --rm -p 4222:4222 -ti nats:latest
```

**终端 2: 启动实时聚合引擎 (ns-engine)**

`ns-engine` 会连接到 NATS，订阅数据并启动内部的并发处理引擎。它的行为由 `configs/config.yaml` 定义。
```sh
go run ./cmd/ns-engine/main.go
```

**终端 3: 启动探针 (ns-probe)**

探针将从指定的网络接口抓包，并发布到 NATS。请将 `<interface_name>` 替换为您的实际网卡名（如 `en0`, `eth0`）。

**注意**: 抓包通常需要管理员权限。
```sh
sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
```

操作正确的话，您将在**终端 2** (`ns-engine`) 中看到 `Manager started with...` 和周期性的 `Starting snapshot...` 日志。

---

## 5. 辅助工具

### 5.1. Gob 解码器

项目提供了一个脚本，用于解码和查看由 `pcap-analyzer` 或 `ns-engine` 生成的 `.dat` 快照文件的内容。

**使用方法**:
```sh
go run ./scripts/gobana/main.go <path_to_dat_file>
```

### 5.2. NATS 消息验证

`ns-probe` 工具内置了订阅者模式，可以用来快速验证 NATS 主题上是否有数据流过。

**使用方法**:
```sh
go run ./cmd/ns-probe/main.go --mode=sub
```
