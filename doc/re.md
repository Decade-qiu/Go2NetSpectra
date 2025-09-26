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
  * **FR-8**: 支持基于五元组的流聚合。
  * **FR-9**: 必须实现以下核心指标的计算：
      * **流量度量**: 包数、字节数、流持续时间。
      * **基数估算**: 使用 **SuperSpread** 算法估算一个流中独立元素的数量（基数），用于发现超级传播者 (Super-Spreader)。
      * **频率估算**: 使用 Count-Min Sketch 估算 Top N 的IP、端口、协议等。
  * **FR-10**: 支持将处理和聚合后的结果写入多个数据存储目标，并可根据配置灵活选择写入器类型（如 Gob 文件、ClickHouse 数据库）。
  * **FR-11**: (高级) 支持可插拔的异常检测算法插件。

#### **2.3. AI 分析模块 (`ns-ai`)**

  * **FR-12**: 提供 gRPC 接口，作为统一的 AI 网关，封装与 LLM 提供商的交互。
  * **FR-13**: 支持通过 gRPC 接收告警信息，并调用 LLM 进行智能分析，生成威胁评估、根因分析和缓解建议。
  * **FR-14**: 支持流式 gRPC 接口，用于通用文本分析和交互式 AI 问答。
  * **FR-15**: 能够通过配置灵活切换 LLM 模型和 `base_url`，支持 OpenAI 兼容的多种 LLM 服务。

#### **2.4. 数据存储模块 (Storage Layer)**

  * **FR-12**: 聚合后的时序指标数据应被写入时序数据库 (如 ClickHouse, VictoriaMetrics)。**目前已集成 ClickHouse。**
  * **FR-13**: 详细的流记录 (Flow Records) 应被写入支持快速检索的分析型数据库 (如 ClickHouse, Elasticsearch)。
  * **FR-14**: 系统的实时状态（如活跃IP列表）应存储在内存数据库中 (如 Redis)。

**2.4. API 与应用模块 (`ns-api`)**

  * **FR-15**: 提供一套 **gRPC API** 用于高性能数据查询，并为 Grafana 等 Web UI 提供兼容的 JSON HTTP 端点。
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

**里程碑 1: 核心引擎与离线分析 (MVP) - ✅ 已完成**

  * **目标**: 验证核心数据处理能力。
  * **核心交付物**:
      * **命令行工具 `pcap-analyzer`**: 可读取 pcap 文件，并启动高性能分析引擎。
      * **并发聚合引擎**: 引擎现在可以根据 `aggregator.types` 配置，**同时加载并运行多种聚合器**（如 `exact` 和 `sketch`）。这使得复杂的联动分析成为可能。
      * **插件化架构 (`Factory` + `TaskGroup`)**: 确立了三层引擎架构。`Manager` 作为并发调度层，`Task` 作为具体的业务逻辑执行层。通过 `TaskGroup` 结构，将特定聚合器类型的所有 `Task` 和 `Writer` 绑定在一起，确保了数据处理的隔离性。
      * **高性能并发设计**: `Manager` 内部采用 **Worker Pool + Channel** 模型，将数据包高效地分发给所有 `Task`。`exact` 任务通过**分片并发**降低锁竞争，`sketch` 任务则通过**无锁原子操作**实现极致性能。
      * **健壮的数据一致性保证**: 通过**优雅停机（Graceful Shutdown）**机制、**只读原子快照**和**周期性重置**机制，确保了数据在每个测量周期内的完整性和准确性。
      * **灵活的写入器配置**: 每个聚合器组都可以独立配置其 `Writer`，每个 `Writer` 也可以拥有独立的写入频率。

**里程碑 2: 实时数据流打通 (Alpha) - ✅ 已完成**

  * **目标**: 实现端到端的实时流量监控原型。
  * **核心交付物**:
      * **统一的 `ns-probe` 命令行工具**: 支持实时抓包 (`probe`) 和 NATS 消息验证 (`sub`)。
      * **实时数据管道**: `probe` 能够从网卡抓包，通过 **Protobuf** 序列化后发布到 **NATS**。`ns-engine` 则订阅数据并送入处理流水线。
      * **ClickHouse 集成**: `ns-engine` 已支持将 `exact` 和 `sketch` 的聚合结果写入不同的 ClickHouse 表中。

**里程碑 3: API 服务与产品化 (Beta) - ✅ 已完成**

  * **目标**: 提供数据查询能力，并使系统易于部署和观察。
  * **核心交付物**:
      * **gRPC 核心 API**: `ns-api` 服务提供 `AggregateFlows`, `TraceFlow`, `QueryHeavyHitters` 等多个高性能查询接口。
      * **智能查询路由**: `ns-api` 能够根据配置初始化多个 `Querier` 实例，并将 gRPC 请求路由到正确的数据源（例如，`AggregateFlows` 查询 `exact` 的数据，`QueryHeavyHitters` 查询 `sketch` 的数据）。
      * **实时预警系统**: 新增 `Alerter` 和 `Notifier` 模块。聚合任务现在可以生成事件，由 `Alerter` 根据规则进行判断，并通过 `Notifier` (如 Webhook) 发送实时告警，满足了 `FR-18` 需求。
      * **AI 智能分析服务 (`ns-ai`)**: 新增 `ns-ai` 微服务，作为统一的 AI 网关，为 `Alerter` 提供智能告警分析能力，并支持交互式 AI 问答。
      * **容器化部署**: 提供了 `docker-compose.yml`，可以一键启动包括 `nats`, `clickhouse`, `ns-engine`, `ns-api`, `ns-ai` 和 `grafana` 在内的完整后端服务。
      * **预置仪表盘**: 提供了一个基础的 Grafana 仪表盘。

  * **下一步**:
      * 为 `ns-api` 增加更完善的日志、监控和错误处理机制。
      * 探索更高级的 Grafana 可视化选项。
      * 开始规划和设计里程碑 4 的分布式能力。
      

**里程碑 4: 分布式与高可用 (Release 1.0)**

  * **目标**: 使系统具备生产环境的可靠性和扩展性。
  * **交付物**:
      * `ns-engine` 和 `ns-api` 实现水平扩展能力。
      * 提供生产级的 Kubernetes 部署脚本 (Helm Chart)。
      * 完成系统的压力测试和稳定性测试。
      * 完善监控和告警机制（系统自身健康度）。

----

### **第三部分：Go 项目结构 (Go Project Structure)**

项目结构遵循 **"Standard Go Project Layout"** 的最佳实践，并已在第一阶段完成重构。

```
netspectra/
├── api/                  # Protobuf 定义, OpenAPI/Swagger YAML 文件
│   └── proto/
│       └── v1/
│           ├── ai.proto
│           └── traffic.proto
├── cmd/                  # 项目主应用入口
│   ├── ns-ai/            # ns-ai 服务的 main.go
│   ├── ns-api/           # ns-api 服务的 main.go
│   ├── ns-engine/        # ns-engine 服务的 main.go
│   ├── ns-probe/         # ns-probe 多功能工具的 main.go
│   └── pcap-analyzer/    # 离线pcap分析工具的 main.go
├── configs/              # 配置文件模板 (config.yaml.example)
├── deployments/          # 部署相关文件
│   ├── docker-compose/
│   └── kubernetes/
├── doc/                  # 项目文档
│   ├── re.md
│   ├── technology.md
│   └── build.md
├── internal/             # 项目内部私有代码，项目核心逻辑
│   ├── ai/               # ns-ai 服务的核心实现
│   ├── config/           # 配置加载
│   ├── engine/           # ns-engine 服务的核心实现
│   │   ├── exacttask/      # 精确统计任务的实现
│   │   ├── manager/        # 引擎的并发调度与生命周期管理
│   │   └── streamaggregator/ # NATS 数据流接入
│   ├── model/            # 核心数据结构与接口 (Packet, Task, Writer)
│   └── probe/            # ns-probe 的核心实现 (Publisher, Subscriber)
├── pkg/                  # 可以被外部应用引用的公共库
│   └── pcap/             # 通用的pcap解析库
├── scripts/              # 辅助脚本
├── test/                 # 测试数据与集成测试
│   └── data/
│       └── test.pcap
├── go.mod
├── go.sum
└── README.md
```

**结构说明** (已更新):

  * **`/cmd`**: 各个二进制文件的启动入口。`ns-probe` 被设计为一个多功能工具。
  * **`/internal`**: 项目的核心私有代码。`engine` 目录被重构为 `manager`+`task` 的清晰架构。
  * **`/pkg`**: 存放可被外部引用的库。
  * **`/doc`**: 存放所有项目文档。
