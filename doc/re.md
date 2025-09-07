### **项目：Go2NetSpectra (网络光谱) 分布式流量监控框架**

### **第一部分：需求规格说明书 (Requirements Specification Document)**

#### **1. 项目概述**

Go2NetSpectra 是一个基于 Go 语言构建的、支持分布式的、高性能网络流量监控与分析框架。项目旨在为网络工程师、安全分析师和系统运维人员提供一个能够对网络流量进行细粒度、多维度深度洞察的平台。它通过高效的数据采集、实时的流处理和强大的数据分析能力，支撑从日常网络性能监控到复杂网络攻击检测等多种上层应用。

**核心价值**:

  * **高性能**: 高吞吐、低延迟、低内存占用，满足大规模网络环境需求。
  * **深度洞察**: 提供从 L2 到 L7 的多维度、细粒度流量指标。
  * **分布式架构**: 所有组件均可水平扩展，确保高可用性和弹性伸缩能力。
  * **实时与离线一体化**: 同时支持实时流量监控和历史流量文件（pcap）的回溯分析。

#### **2. 功能性需求 (Functional Requirements)**

**2.1. 数据采集模块 (`ns-probe`)**

  * **FR-1**: 支持从网络接口实时捕获数据包 (依赖 `eBPF` 或 `AF_PACKET`)。
  * **FR-2**: 支持读取并解析标准 `pcap`/`pcapng` 格式的离线文件。
  * **FR-3**: 支持对捕获的流量进行初步解析，提取关键元数据（如五元组、时间戳、包长）。
  * **FR-4**: 支持将解析后的数据以高效的序列化格式 (如 Protobuf) 发送到数据传输层。
  * **FR-5**: 采集探针必须是轻量级的，对宿主机的资源消耗要尽可能低。

**2.2. 数据处理模块 (`ns-engine`)**

  * **FR-6**: 支持从数据传输层（消息队列）消费数据。
  * **FR-7**: 能够对数据进行深度的协议解析，至少包括：Ethernet, IPv4/v6, TCP, UDP, ICMP, DNS, HTTP/1.x。
  * **FR-8**: 支持基于五元组的流聚合与生命周期管理（Active, Timeout, Closed）。
  * **FR-9**: 必须实现以下核心指标的计算：
      * **流量度量**: 包数、字节数、流持续时间。
      * **基数估算**: 使用 HyperLogLog 估算独立源/目的IP数、独立端口数等。
      * **频率估算**: 使用 Count-Min Sketch 估算 Top N 的IP、端口、协议等。
  * **FR-10**: 支持将处理和聚合后的结果写入多个数据存储目标。
  * **FR-11**: (高级) 支持可插拔的异常检测算法插件。

**2.3. 数据存储模块 (Storage Layer)**

  * **FR-12**: 聚合后的时序指标数据应被写入时序数据库 (如 ClickHouse, VictoriaMetrics)。
  * **FR-13**: 详细的流记录 (Flow Records) 应被写入支持快速检索的分析型数据库 (如 ClickHouse, Elasticsearch)。
  * **FR-14**: 系统的实时状态（如活跃IP列表）应存储在内存数据库中 (如 Redis)。

**2.4. API 与应用模块 (`ns-api`)**

  * **FR-15**: 提供一套 RESTful 或 GraphQL API，用于数据查询。
  * **FR-16**: API 必须支持按时间范围、一个或多个维度（IP, 端口, 协议等）进行组合查询和过滤。
  * **FR-17**: 提供管理接口，用于配置采集规则和告警阈值。
  * **FR-18**: 支持配置告警规则，并在触发时通过 Webhook 等方式发送通知。

**2.5. 可视化模块 (Presentation Layer)**

  * **FR-19**: 提供与 Grafana 集成的能力，并预置一套覆盖核心监控指标的仪表盘模板。

#### **3. 非功能性需求 (Non-Functional Requirements)**

  * **NFR-1 (性能)**:
      * **吞吐量**: 单个 `ns-engine` 节点应能处理至少 5Gbps 或 50万 PPS 的流量。整个集群可通过水平扩展线性提升处理能力。
      * **延迟**: 数据从被采集到在仪表盘上可查询的端到端延迟应低于 5 秒。
  * **NFR-2 (可靠性)**:
      * 核心服务 (`ns-engine`, `ns-api`) 必须是无状态的，支持多实例部署以实现高可用。
      * 数据传输和存储层应采用具备高可用性的集群方案（如 Kafka 集群, ClickHouse 集群）。
  * **NFR-3 (可扩展性)**:
      * 所有组件都必须能够独立地进行水平扩展。
      * 系统设计应支持插件化，方便未来扩展新的协议解析器和分析算法。
  * **NFR-4 (可维护性)**:
      * 所有组件都应提供详细的日志记录和健康检查端点。
      * 提供容器化部署方案（Docker, Kubernetes）。
  * **NFR-5 (安全性)**:
      * API 接口需要有认证和授权机制。
      * 组件间的通信应支持 TLS 加密。

-----

### **第二部分：项目计划 (Development Process & Plan)**

**里程碑 1: 核心引擎与离线分析 (MVP) - (预计 4-6 周)**

  * **目标**: 验证核心数据处理能力。
  * **交付物**:
      * 一个命令行工具 (`ns-cli`)，可以读取 pcap 文件。
      * 实现 `ns-engine` 的核心库：协议解析、流聚合、核心指标计算。
      * 数据能够成功写入本地 ClickHouse 实例。
      * 一份基础的 Grafana Dashboard，可以展示 pcap 文件的分析结果。
      * 完成核心库的单元测试。

**里程碑 2: 实时数据流打通 (Alpha) - (预计 4-6 周)**

  * **目标**: 实现端到端的实时流量监控。
  * **交付物**:
      * `ns-probe` 服务，能从网卡抓包并将数据发送到 Kafka。
      * `ns-engine` 服务，能从 Kafka 消费数据并处理。
      * 搭建起 Kafka, ClickHouse, Grafana 的基础环境。
      * Grafana 仪表盘可以准实时地（秒级延迟）展示流量数据。

**里程碑 3: API 服务与产品化 (Beta) - (预计 3-5 周)**

  * **目标**: 提供数据查询能力，并使系统易于部署。
  * **交付物**:
      * `ns-api` 服务，提供按维度查询流量指标的 RESTful API。
      * 为所有服务 (`probe`, `engine`, `api`) 编写 K8S 部署文件。
      * 提供一份 K8S + Helm 的部署指南。
      * 完善的文档：README, API 文档, 部署指南。

**里程碑 4: 分布式与高可用 (Release 1.0) - (预计 4-6 周)**

  * **目标**: 使系统具备生产环境的可靠性和扩展性。
  * **交付物**:
      * `ns-engine` 和 `ns-api` 实现水平扩展能力。
      * 提供 Kubernetes 部署脚本 (Helm Chart)。
      * 完成系统的压力测试和稳定性测试。
      * 完善监控和告警机制（系统自身健康度）。

-----

### **第三部分：Go 项目结构建议 (Go Project Structure)**

推荐采用经典的 **"Standard Go Project Layout"** 变体，它能很好地组织项目，实现关注点分离。

```
netspectra/
├── api/                  # Protobuf 定义, OpenAPI/Swagger YAML 文件
│   └── proto/
│       └── v1/
│           └── traffic.proto
├── cmd/                  # 项目主应用入口
│   ├── ns-api/           # ns-api 服务的 main.go
│   │   └── main.go
│   ├── ns-engine/        # ns-engine 服务的 main.go
│   │   └── main.go
│   └── ns-probe/         # ns-probe 服务的 main.go
│       └── main.go
├── configs/              # 配置文件模板 (config.yaml.example)
├── deployments/          # 部署相关文件
│   ├── docker-compose/
│   │   └── docker-compose.yml
│   └── kubernetes/
│       ├── ns-api-deployment.yaml
│       └── ...
├── internal/             # 项目内部私有代码，项目核心逻辑
│   ├── api/              # ns-api 服务的实现 (HTTP handlers, routes)
│   ├── engine/           # ns-engine 服务的实现
│   │   ├── pipeline/     # 数据处理流水线
│   │   ├── protocol/     # 协议解析器
│   │   └── storage/      # 数据库写入逻辑
│   ├── probe/            # ns-probe 服务的实现
│   │   ├── capture/      # 抓包逻辑 (eBPF, AF_PACKET)
│   │   └── publisher/    # 数据发布到 Kafka 的逻辑
│   └── pkg/              # 项目内部共享的包
│       ├── config/       # 配置加载
│       ├── logger/       # 日志封装
│       └── probestore/   # 概率数据结构 (HLL, Count-Min Sketch)
├── pkg/                  # 可以被外部应用引用的公共库 (初期可少用或不用)
│   └── pcap/             # 例如，一个通用的pcap解析库
├── scripts/              # 辅助脚本 (构建、测试、部署等)
│   ├── build.sh
│   └── lint.sh
├── test/                 # 集成测试、端到端测试和测试数据
│   └── data/
│       └── test.pcap
├── go.mod                # Go 模块文件
├── go.sum
└── README.md
```

**结构说明**:

  * **`/cmd`**: 清晰地分离了不同服务的启动入口，每个子目录都是一个独立的 `main` 包。
  * **`/internal`**: 这是项目的核心。Go 编译器会保证 `internal` 目录下的代码只能被其父目录及同级目录的代码引用。这强制实现了良好的封装，防止项目内部逻辑被外部不期望地依赖。
      * `internal/api`, `internal/engine`, `internal/probe` 分别对应三个核心服务，实现了服务间的逻辑解耦。
      * `internal/pkg` 用于存放项目内部的公共代码，避免重复造轮子。
  * **`/pkg`**: 用于存放可以安全地被外部项目引用的代码。如果 NetSpectra 打算提供一个 SDK 供其他应用使用，代码就应放在这里。在项目初期，可以优先使用 `internal`。
  * **`/api`**: 集中管理所有 API 定义，无论是 gRPC/Protobuf 还是 REST/OpenAPI，都让接口定义清晰可见。
  * **`/deployments`**: 将部署配置与应用代码分离，便于 DevOps 维护。
  * **`/scripts`**: 自动化常用操作，提高开发效率。