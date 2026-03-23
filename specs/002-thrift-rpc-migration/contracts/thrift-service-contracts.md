# Contract: Thrift Service Contracts

## Source Of Truth

- **IDL root**: `api/thrift/v1/`
- **Generated Go root**: `api/gen/thrift/v1/`
- **Legacy contract roots to retire from supported workflows**:
  - `api/proto/v1/`
  - `api/gen/v1/`

The Thrift IDL becomes the only supported source of truth for packet transport,
query RPC, and AI RPC after the cutover.

## Packet Transport Contract

### Required structs

- `FiveTuple`
  - `src_ip`
  - `dst_ip`
  - `src_port`
  - `dst_port`
  - `protocol`
- `PacketInfo`
  - `timestamp`
  - `five_tuple`
  - `length`

### Contract rules

- The packet payload carried over NATS must serialize the same packet semantics
  as the current maintained path.
- The manager ingress may decode Thrift payloads into internal domain models,
  but supported packet publishers and consumers must agree on one Thrift
  contract version.
- Mixed Protobuf and Thrift packet payloads are unsupported after cutover and
  must fail explicitly during decode rather than silently falling back.

## Query Service Contract

### Supported RPC methods

- `HealthCheck() -> HealthCheckResponse`
- `SearchTasks() -> SearchTasksResponse`
- `AggregateFlows(AggregationRequest) -> QueryTotalCountsResponse`
- `TraceFlow(TraceFlowRequest) -> TraceFlowResponse`
- `QueryHeavyHitters(HeavyHittersRequest) -> HeavyHittersResponse`

### Request/response expectations

- `AggregationRequest` continues to support `task_name`, `end_time`, and
  optional flow filters when those filters are part of the supported query path.
- `TraceFlowRequest` continues to carry a `task_name`, a flow-key map, and an
  `end_time`.
- `HeavyHittersRequest` continues to carry a task name, heavy-hitter type,
  time boundary, and result limit.
- Aggregate, trace, and heavy-hitter responses preserve the current supported
  business semantics even though the wire format changes.

### HTTP compatibility rule

Supported HTTP/JSON handlers must map JSON bodies to internal query DTOs or
Thrift adapter inputs without depending on Protobuf-only helpers such as
`protojson`. JSON support remains an API concern, not the source of truth for
query contracts.

## AI Service Contract

### Supported unary RPC methods

- `AnalyzeTraffic(AnalyzeTrafficRequest) -> AnalyzeTrafficResponse`

This method remains a direct request/response contract used by alerting and
operator-triggered AI analysis.

### Supported incremental prompt-analysis workflow

Because the old `AnalyzePromptStream` server-streaming gRPC method has no
native Thrift equivalent, the new contract is modeled as three RPCs:

- `StartPromptAnalysis(PromptAnalysisRequest) -> PromptAnalysisSession`
- `ReadPromptChunks(PromptChunkRequest) -> PromptChunkResponse`
- `CancelPromptAnalysis(PromptCancelRequest) -> PromptCancelResponse`

### Session rules

- `StartPromptAnalysis` returns a `session_id` and enough metadata for the
  client to begin polling.
- `ReadPromptChunks` blocks until at least one chunk or a terminal state is
  available, then returns zero or more chunks plus a terminal flag or terminal
  error when the session completes or fails.
- `CancelPromptAnalysis` is idempotent and may be called by clients or during
  service shutdown.
- Idle prompt sessions expire after 10 minutes by default.
- Service shutdown cancels all active prompt sessions before the RPC listener
  exits.
- Session lifecycle, retention, and cleanup behavior must be documented and
  verified because this feature introduces new goroutine and buffer ownership.

## Mixed-Version Failure Expectations

- A Thrift-only runtime does not attempt to serve legacy gRPC clients.
- Legacy Protobuf packet payloads sent into maintained packet consumers must be
  rejected during decode.
- The retired `ns-api/v1` entrypoint must fail fast with an explicit
  unsupported-workflow error instead of silently serving the old HTTP API.

## Breaking Change Notes

- Old gRPC stubs and Proto message types are retired from supported runtime use.
- Repository-provided CLI scripts must migrate to the new Thrift client flow.
- Existing business capabilities remain supported, but the wire contract and
  some client interaction details change as part of the approved breaking cutover.
