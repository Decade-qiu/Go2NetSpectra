# Data Model: Repository-Wide Thrift Contract Migration

## Entity: ContractArtifactSet

**Purpose**: Represents the authoritative Thrift contract inputs and the
generated Go outputs derived from them.

**Fields**:

- `idl_root`: The directory containing the source `.thrift` files.
- `generated_root`: The directory containing generated Go bindings.
- `services`: The contract groups covered by the artifact set, such as packet,
  query, and AI.
- `generator_command`: The supported regeneration command or workflow.
- `legacy_paths`: Retired Proto/gRPC paths replaced by this artifact set.
- `status`: Planned, generated, validated, or retired.

**Validation Rules**:

- `idl_root` and `generated_root` must each identify exactly one supported
  source-of-truth location.
- Every active service contract must belong to exactly one artifact set.
- `legacy_paths` must be explicit for a breaking cutover.

**Relationships**:

- Owns one or more `PacketTransportMessage`, `QueryOperation`, and
  `PromptAnalysisSession` contract definitions.
- Is referenced by `CutoverSurface` records in deployment, script, and doc
  updates.

**State Transitions**:

- `Planned -> Generated -> Validated -> Active`
- `Active -> Retired` for legacy Proto/gRPC artifacts at the end of cutover

## Entity: PacketTransportMessage

**Purpose**: Defines the normalized packet metadata exchanged between probe,
offline readers, NATS consumers, and the manager ingress.

**Fields**:

- `timestamp`: Packet capture time.
- `src_ip`: Source IP bytes.
- `dst_ip`: Destination IP bytes.
- `src_port`: Source port.
- `dst_port`: Destination port.
- `protocol`: Layer-4 protocol number.
- `length`: Packet length in bytes.

**Validation Rules**:

- `timestamp` must be present for supported packet flows.
- Five-tuple fields must be complete before a message is accepted by the
  manager ingress.
- Encoded and decoded values must preserve the same semantics as the current
  maintained packet path.

**Relationships**:

- Produced by probe publishers and offline readers.
- Consumed by NATS subscribers, stream aggregators, and manager ingress.
- Maps to internal `model.PacketInfo`.

## Entity: QueryOperation

**Purpose**: Represents one supported query capability exposed over Thrift RPC
and optionally mirrored through supported HTTP endpoints.

**Fields**:

- `kind`: HealthCheck, SearchTasks, AggregateFlows, TraceFlow, or
  QueryHeavyHitters.
- `task_name`: Task selector when applicable.
- `end_time`: Query time boundary when applicable.
- `filters`: Optional task or flow filters.
- `limit`: Result cap when applicable.
- `backend`: Exact or sketch query backend expected to satisfy the operation.
- `result_type`: The response shape returned for the operation.

**Validation Rules**:

- `kind` must map to exactly one supported contract method.
- `filters` must use supported flow-key names when trace semantics apply.
- `backend` must reflect an existing query implementation path.

**Relationships**:

- Produces one or more `QueryResultRecord` instances.
- Is served by the query RPC adapter and optionally by HTTP handlers.
- Reads data from ClickHouse-backed query implementations.

## Entity: QueryResultRecord

**Purpose**: Captures a response unit returned by aggregate, trace, or
heavy-hitter operations.

**Fields**:

- `result_kind`: TaskSummary, FlowLifecycle, or HeavyHitter.
- `task_name`: Task context when applicable.
- `flow`: Flow identifier or filter summary for heavy hitters.
- `total_packets`: Packet count metric.
- `total_bytes`: Byte count metric.
- `flow_count`: Number of flows for aggregate responses.
- `first_seen`: First observation time for flow traces.
- `last_seen`: Last observation time for flow traces.
- `value`: Ranked heavy-hitter metric value.

**Validation Rules**:

- The populated fields must match `result_kind`.
- Count and size metrics must preserve current supported query semantics.
- Trace lifecycle timestamps must remain ordered.

**Relationships**:

- Returned by `QueryOperation`.
- Serialized through Thrift RPC responses and optionally JSON HTTP responses.

## Entity: PromptAnalysisSession

**Purpose**: Tracks a long-lived AI prompt analysis that emits output in chunks
instead of a single unary response.

**Fields**:

- `session_id`: Unique identifier returned to the client.
- `prompt`: The user-submitted prompt.
- `chunks`: Buffered output chunks available for retrieval.
- `done`: Whether generation has completed.
- `error_text`: Terminal failure message when the session fails.
- `created_at`: Session creation time.
- `last_activity_at`: Most recent read or write time.
- `cancel_reason`: Optional cancellation cause.

**Validation Rules**:

- `session_id` must be unique among live sessions.
- Sessions must eventually reach a terminal state: completed, canceled, failed,
  or expired.
- Chunk buffers must have bounded growth and explicit cleanup rules.

**Relationships**:

- Created by an AI prompt-start RPC call.
- Read by one or more chunk retrieval calls.
- Owned by the AI service runtime and cleaned up on cancellation or shutdown.

**State Transitions**:

- `Created -> Running -> Completed`
- `Created/Running -> Canceled`
- `Created/Running -> Failed`
- `Completed/Canceled/Failed -> Expired`

## Entity: CutoverSurface

**Purpose**: Records a code, config, script, document, or deployment surface
that must move from Proto/gRPC semantics to Thrift semantics in the same
delivery.

**Fields**:

- `path`: Repository path or runtime surface name.
- `surface_type`: Code, config, script, doc, deployment, or generated artifact.
- `current_protocol`: Current Proto/gRPC behavior or naming.
- `target_protocol`: Target Thrift/RPC behavior or naming.
- `migration_action`: Replace, rename, remove, or mark retired.
- `verification_signal`: The check proving the surface is synchronized.

**Validation Rules**:

- Every supported runtime surface touched by the migration must have a
  corresponding cutover record.
- `migration_action` must be explicit for breaking changes.
- `verification_signal` must identify an executable or inspectable proof.

**Relationships**:

- References `ContractArtifactSet`.
- Drives tasks across code, docs, config, and deployment assets.
