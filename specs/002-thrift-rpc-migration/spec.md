# Feature Specification: Repository-Wide Thrift Contract Migration

**Feature Branch**: `002-thrift-rpc-migration`  
**Created**: 2026-03-23  
**Status**: Ready for Planning  
**Input**: User description: "帮我将项目里的protocol buffer序列化协议完全换成thrift"

## Clarifications

### Session 2026-03-24

- Q: HTTP/JSON 入口是否继续作为受支持接口？ → A: 仅保留 Grafana 兼容 HTTP 查询入口，退役 legacy HTTP API。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Cut Over Packet Transport Contracts (Priority: P1)

作为维护实时抓包与离线分析链路的开发者，我希望系统内部用于传输包信息的统一契约从
Protocol Buffer 切换为 Thrift，这样 probe、pcap 分析、NATS 消息消费和聚合入口
不再依赖两套不同的序列化体系，也不会继续围绕旧契约扩展。

**Why this priority**: 运行时数据入口是整个系统的基础。如果包传输链路仍保留旧契约，
后续服务接口的替换只会留下一个更难维护的“双协议”状态。

**Independent Test**: 只验证抓包发布、离线读取、NATS 订阅和聚合入口这条链路，
在不启动查询或 AI 服务的情况下，仍应能够端到端完成 Thrift 编码、传输、解码和处理。

**Acceptance Scenarios**:

1. **Given** probe 与 engine 都已升级到新的契约版本，**When** probe 发布实时抓包数据到消息总线，**Then**
   engine 能够使用 Thrift 契约解码并继续后续处理，而不需要 Protocol Buffer 解码路径。
2. **Given** 离线 pcap 分析工作流使用共享的包传输契约，**When** 它将解析结果送入后续处理路径，**Then**
   关键字段语义与迁移前保持一致，包括时间戳、地址、端口、协议标识和长度信息。
3. **Given** 运行中的新系统收到旧的 Protocol Buffer 消息，**When** 该消息进入已迁移的处理入口，**Then**
   系统会明确拒绝或报告不支持的旧流量，而不是静默接受或错误解析。

---

### User Story 2 - Replace Query And AI Service RPC (Priority: P2)

作为运行查询服务、告警链路和 AI 分析能力的维护者，我希望现有基于 Protobuf/gRPC 的服务契约
整体切换为 Thrift/Thrift RPC，这样仓库内维护的服务间调用和仓库自带客户端脚本都基于同一种契约体系。

**Why this priority**: 查询和 AI 服务是当前仓库最直接的对外运行面。如果 RPC 层不一起切换，
项目仍然需要长期维护 Protobuf/gRPC 生成物和相关操作说明。

**Independent Test**: 仅启动升级后的查询服务和 AI 服务，并使用仓库内自带脚本或等价调用路径，
验证关键业务操作可通过 Thrift RPC 完成，不依赖旧的 gRPC 客户端。

**Acceptance Scenarios**:

1. **Given** 查询服务已升级，**When** 维护者通过支持的客户端路径执行健康检查、任务检索、
   聚合查询、流追踪或 heavy hitter 查询，**Then** 请求与响应通过 Thrift RPC 完成，并返回正确的业务结果。
2. **Given** AI 服务已升级，**When** 维护者通过支持的客户端路径触发分析请求或流式分析请求，**Then**
   业务能力保持可用，且调用方不再依赖 Protobuf/gRPC 契约。
3. **Given** 仓库选择一次性破坏性切换，**When** 旧版 gRPC 客户端连接到新服务，**Then**
   服务会快速、明确地失败，提醒调用方必须整体升级，而不是进入未定义兼容状态。

---

### User Story 3 - Retire Legacy Protocol Operations Surface (Priority: P3)

作为负责构建、部署和日常维护的工程师，我希望文档、脚本、配置、生成物目录和部署资产
都同步收敛到 Thrift 契约，只保留 Grafana 兼容 HTTP 查询入口作为受支持的非 RPC 入口，
并退役 legacy HTTP API，不再把 Protocol Buffer 或 gRPC 作为受支持的当前工作流，
这样团队不会在后续开发和运维中误用已经淘汰的协议栈。

**Why this priority**: 即使代码路径已切换，如果文档、脚本和部署资产仍指向旧协议，
团队仍会在构建、排障和新成员上手时反复踩坑。

**Independent Test**: 仅检查仓库的受支持操作面，包括构建说明、契约生成说明、服务启动说明、
部署资产、仓库脚本和保留的 Grafana 兼容 HTTP 查询入口，确认它们已统一指向 Thrift 工作流，
并移除对旧协议栈和 legacy HTTP API 的当前依赖。

**Acceptance Scenarios**:

1. **Given** 维护者按照仓库文档搭建和运行系统，**When** 他执行受支持的构建、生成和启动步骤，**Then**
   不需要安装或执行当前工作流所依赖的 Protocol Buffer/gRPC 工具链。
2. **Given** 维护者查看受支持的配置与部署资产，**When** 他定位服务监听、契约生成或依赖说明，**Then**
   不会再把 Protobuf/gRPC 误认为当前必须维护的运行面。
3. **Given** 仓库内仍保留历史提交和旧文档痕迹，**When** 维护者区分当前支持与历史信息，**Then**
   当前受支持路径会被明确标识为 Thrift，而历史内容不会继续误导新的变更和运维流程。

### Edge Cases

- 新版本运行时收到旧的 Protocol Buffer 消息或旧的 gRPC 请求时，必须快速失败并提供可识别的版本不匹配信号。
- 实时抓包、离线分析和脚本调用都依赖时间戳、地址、端口、协议号和计数字段；迁移后这些字段的语义不能因契约替换而漂移。
- 如果查询或 AI 能力存在流式或多次返回语义，迁移后仍需保留等价的业务行为，而不是无提示退化成功能缺失。
- 保留的 Grafana 兼容 HTTP 查询入口不能继续隐藏依赖于 Protobuf 生成类型的当前支持路径，退役的 legacy HTTP API 需要有明确的下线路径说明。
- 生成物、文档与部署资产中若同时残留新旧协议信息，团队可能误判真实支持面；规格必须把这类“半迁移”视为未完成状态。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST 将当前维护中的内部传输契约从 Protocol Buffer 全量切换为 Thrift，
  覆盖实时抓包、离线分析、消息总线传输和进入聚合入口的受支持路径。
- **FR-002**: System MUST 将当前维护中的查询服务和 AI 服务对外 RPC 契约从 Protobuf/gRPC
  全量切换为 Thrift/Thrift RPC。
- **FR-003**: System MUST 在协议替换后保持当前受支持业务能力可用，
  包括包传输、任务查询、聚合查询、流追踪、heavy hitter 查询和 AI 分析相关操作。
- **FR-004**: System MUST 以一次性破坏性切换的方式完成迁移，
  且 MUST NOT 要求新旧协议在生产支持范围内长期并行或互通。
- **FR-005**: System MUST 明确拒绝旧的 Protocol Buffer 消息和旧的 gRPC 调用，
  使调用方能够快速识别版本不匹配，而不是得到静默错误结果。
- **FR-006**: System MUST 保持关键数据语义稳定，包括时间戳、五元组、长度、计数、
  生命周期信息以及当前受支持查询结果中的核心字段含义。
- **FR-007**: System MUST 为仓库内自带的客户端脚本、服务入口和支持的操作路径
  提供与新 Thrift 契约一致的调用方式。
- **FR-008**: System MUST 同步更新当前受支持的构建说明、契约生成说明、脚本、
  文档、配置说明和部署资产，移除对 Protocol Buffer/gRPC 的当前依赖描述。
- **FR-009**: Users MUST be able to 明确识别新的 Thrift 契约来源位置，
  并据此判断后续修改应落在哪些契约、生成物和服务入口上。
- **FR-010**: System MUST 将“半迁移”视为不合格状态；只要受支持路径中仍要求维护者
  依赖 Protobuf/gRPC 完成当前工作流，迁移就不能视为完成。
- **FR-011**: System MUST 保留 Grafana 兼容 HTTP 查询入口作为唯一受支持的非 RPC 对外入口，
  并退役 legacy HTTP API；保留的 HTTP 入口 MUST 脱离对 Protobuf 专用类型和序列化行为的当前依赖。

### Contract & Configuration Impact *(mandatory when applicable)*

- **CC-001**: 本次迁移必须识别并替换当前由 `api/proto/v1/`、`api/gen/v1/`
  以及相关客户端、服务端、脚本所承载的受支持契约，并明确新的 Thrift 契约来源和生成物位置。
- **CC-002**: 任何当前配置、环境变量、部署资产或操作文档中对 gRPC 监听、
  Protobuf 生成或相关工具链的依赖，都必须在同一交付中改为 Thrift 语义或被明确宣布退役。
- **CC-003**: 由于本次为破坏性切换，发布与运维说明必须明确要求所有受支持服务、
  仓库自带客户端和自动化脚本整体升级；新旧协议混跑不在支持范围内。
- **CC-004**: 如果某些存量查询结果、存储结构或外部输入输出语义会受到契约替换影响，
  规格必须记录哪些行为保持稳定，哪些行为允许作为破坏性变更一并调整。

### Key Entities *(include if feature involves data)*

- **Packet Transport Contract**: 定义实时抓包、离线分析和消息总线之间交换的统一包数据契约。
- **Query Service Contract**: 定义查询服务暴露的健康检查、任务检索、聚合、追踪和 heavy hitter 能力。
- **AI Service Contract**: 定义 AI 分析服务暴露的请求、响应和流式交互语义。
- **Migration Surface Inventory**: 列出所有需要从 Protobuf/gRPC 收敛到 Thrift 的受支持代码路径、
  脚本、文档、配置和部署资产。
- **Cutover Boundary**: 描述本次破坏性切换允许打破的兼容性范围，以及明确不支持的混跑场景。

## Architecture & Operational Impact *(mandatory when applicable)*

### Pipeline Impact

- **Capture/Parse**: 实时抓包发布、离线 pcap 读取和共享包编码/解码路径都要切换到 Thrift 契约，
  并保持包字段语义与当前受支持工作流一致。
- **Aggregation/Storage**: 聚合入口和后续处理路径必须消费新的 Thrift 契约输入；
  若契约替换影响到任务结果、写出格式或查询语义，需要在规格中显式说明。
- **Query/API/AI**: 查询服务、AI 服务、仓库自带客户端脚本以及保留的 Grafana 兼容 HTTP 查询入口
  都必须摆脱对 Protobuf/gRPC 契约的当前依赖，并统一收敛到新的服务契约体系；legacy HTTP API 在本次迁移中退役。
- **Deployment/Config**: 构建步骤、生成步骤、运行参数、部署清单和运维文档必须同步体现
  “Thrift 是当前唯一受支持契约”的事实，避免留下双轨操作说明。

### Verification Plan

- **VP-001**: 验证实时抓包、离线分析、消息总线消费和聚合入口的代表性链路，
  证明包传输已基于 Thrift 契约完成端到端处理。
- **VP-002**: 验证查询服务和 AI 服务的关键业务操作可以通过新的 RPC 契约完成，
  且仓库内自带脚本或等价调用路径能够成功运行。
- **VP-003**: 验证受支持的构建、契约生成、文档和部署流程不再要求维护者安装、
  生成或运行当前工作流所依赖的 Protobuf/gRPC 工具链。
- **VP-004**: 验证旧消息和旧客户端在切换后会得到明确失败信号，
  使整体升级要求对调用方和运维方可见。

## Scope Boundaries

### In Scope

- 当前仓库中仍被维护和支持的 Protocol Buffer 序列化契约替换。
- 当前仓库中仍被维护和支持的 gRPC 查询与 AI 服务契约替换。
- 与此次协议迁移直接相关的生成物、脚本、文档、配置说明和部署资产同步更新。
- 保留 Grafana 兼容 HTTP 查询入口并退役 legacy HTTP API。
- 仓库内自带客户端、服务入口和受支持工作流的整体切换与验证。
- 明确破坏性切换的边界、失败方式和运维升级要求。

### Out of Scope

- 为外部未知调用方提供长期双协议兼容或并行运行方案。
- 借本次迁移额外引入与协议替换无关的新产品功能。
- 与当前维护路径无关的历史目录、历史提交或纯归档内容全面重写。
- 对业务语义、存储策略或查询能力做与协议迁移无直接关系的大幅改造。
- 继续维护 legacy HTTP API 作为受支持接口。

## Assumptions & Dependencies

- 用户已确认本次需求包含完整的 RPC 体系替换，而不是仅替换消息序列化格式。
- 用户已确认接受一次性破坏性切换，不要求新旧 Protocol Buffer/gRPC 与 Thrift/Thrift RPC 并行兼容。
- 仓库内自带脚本、服务入口和文档是本次迁移的主要支持面；未在仓库内维护的外部调用方需要自行跟进升级。
- 本次迁移的目标是替换契约体系而不是改变系统核心业务能力，因此默认要求受支持功能语义保持稳定。
- 如果某些当前接口或脚本被判定不再继续支持，必须在规格、计划和交付说明中显式声明，而不能通过遗漏实现“自然下线”。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 维护者能够通过受支持的实时或离线运行路径完成一次端到端包处理流程，
  且该流程不再依赖 Protocol Buffer 作为当前运行契约。
- **SC-002**: 维护者能够通过仓库内受支持的客户端路径完成查询服务和 AI 服务的关键调用，
  且这些调用全部通过新的 RPC 契约完成。
- **SC-003**: 当前受支持的构建、契约生成、启动和部署说明中，不再存在要求维护者安装、
  执行或依赖 Protobuf/gRPC 工具链才能完成工作流的步骤。
- **SC-004**: 对于切换后的旧协议请求或消息，系统能够在一次调用或一次消息处理内给出明确失败信号，
  不出现静默兼容或错误结果被当作成功处理的情况。
- **SC-005**: 维护者能够在 5 分钟内从当前仓库中定位新的契约来源、受影响服务入口和操作文档，
  而无需再把旧的 Protobuf/gRPC 资产当作当前支持面的事实来源。
