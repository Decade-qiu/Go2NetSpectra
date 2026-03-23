# Research: Repository-Wide Thrift Contract Migration

## Decision 1: Use `api/thrift/v1` as the new contract source of truth

**Decision**: Introduce Thrift IDL files under `api/thrift/v1` and generate Go
bindings under `api/gen/thrift/v1`, treating those outputs as derived artifacts
in the same way `api/gen/v1` is handled today.

**Rationale**: The repository already treats contracts as a dedicated boundary.
Putting Thrift into an explicit new root makes the cutover visible, avoids
mixing legacy and new definitions in the same directory, and allows the plan to
retire Proto/gRPC assets cleanly once the migration is complete.

**Alternatives considered**:

- **Overwrite `api/proto/v1` in place**: Rejected because it would blur which
  files are legacy versus current and make operational cleanup harder to review.
- **Store Thrift files inside `internal/`**: Rejected because contracts are a
  public/runtime boundary, not private implementation detail.
- **Reuse `api/gen/v1` for Thrift outputs**: Rejected because it would hide the
  generator change and invite accidental mixed imports.

## Decision 2: Use Thrift binary protocol consistently across NATS and RPC

**Decision**: Standardize on Thrift binary protocol for generated structs,
using serializer/deserializer helpers for NATS packet payloads and buffered TCP
transport for query/AI RPC services.

**Rationale**: Apache Thrift documents transport and protocol as separate
concerns, and its tutorial uses buffered transport with binary protocol for
client/server communication. Reusing one protocol family across packet payloads
and RPC reduces debugging complexity and keeps the on-wire format explicit.

**Alternatives considered**:

- **Use compact protocol everywhere**: Rejected as the default because it adds
  another optimization variable before baseline behavior is proven. It can be
  revisited later if validation shows binary protocol is a throughput problem.
- **Use different protocols for NATS and RPC**: Rejected because mixed protocol
  choices increase troubleshooting and configuration complexity.
- **Use HTTP transport as the primary RPC path**: Rejected because the current
  runtime uses direct RPC listeners, and buffered TCP is the closest structural
  replacement for the gRPC role.

## Decision 3: Move business logic onto internal models instead of generated wire structs

**Decision**: Keep generated Thrift structs at package edges and route runtime
logic through internal/domain models:

- `internal/probe` and `internal/engine/*` should exchange `model.PacketInfo`
  or equivalent domain packet values after decoding.
- `internal/query` should consume internal query DTOs and return internal
  result models, with Thrift/HTTP adapters mapping at the API layer.

**Rationale**: Today the manager, query layer, and HTTP handlers directly use
generated Protobuf types. Repeating that coupling with generated Thrift types
would solve the immediate cutover but preserve the same long-term maintenance
problem. Introducing adapter seams keeps serialization concerns inside transport
packages and makes the HTTP endpoints independent from generated contract code.

**Alternatives considered**:

- **Replace Protobuf types with generated Thrift types everywhere**: Rejected
  because it preserves transport leakage deep inside business logic.
- **Maintain dual Proto and Thrift adapters for a long transition**: Rejected
  because the user approved a breaking cutover rather than a compatibility
  window.

## Decision 4: Replace AI server-streaming with session-based chunk retrieval

**Decision**: Replace the gRPC `AnalyzePromptStream` server-streaming RPC with
an explicit Thrift session workflow:

1. Start prompt analysis and receive a session identifier.
2. Poll or read available chunks for that session.
3. Cancel or expire the session when complete or abandoned.

**Rationale**: Apache Thrift service definitions expose unary request/response
methods plus `oneway` calls, but they do not provide a native gRPC-style
server-streaming primitive. A session-based chunk API preserves incremental
delivery semantics while staying inside a Thrift-only contract surface.

**Alternatives considered**:

- **Collapse the feature to one unary `AnalyzePrompt` response**: Rejected
  because the approved spec keeps prompt streaming as a supported behavior.
- **Keep gRPC only for streaming**: Rejected because it would violate the goal
  of removing Proto/gRPC from supported workflows.
- **Add a separate HTTP/SSE or WebSocket channel**: Rejected because it would
  split the service surface across protocols and complicate deployment.

## Decision 5: Rename protocol-specific runtime keys to neutral RPC names

**Decision**: Rename active runtime surfaces such as `grpc_listen_addr` and
deployment port labels from `grpc` to neutral `rpc` naming where they describe
service roles rather than file formats. Keep `thrift` terminology for contract
directories, code generation, and migration documentation.

**Rationale**: The user accepted a breaking cutover, so this is the right time
to remove protocol-specific naming drift from runtime configuration. Using
neutral RPC names avoids baking the next protocol migration into config again,
while Thrift-specific terms remain explicit where they actually matter.

**Alternatives considered**:

- **Keep `grpc_*` names for compatibility**: Rejected because it would leave
  misleading operational semantics immediately after removing gRPC.
- **Rename all runtime settings to `thrift_*`**: Rejected because the bind
  address and service port describe an RPC role, not a source-file format.

## Decision 6: Validate the cutover with a strict migration ladder

**Decision**: Require the following evidence before considering the feature
complete:

1. Thrift IDL generation succeeds and the generated outputs are committed.
2. Focused package tests pass for packet codec, manager, stream aggregator,
   query, AI, and alerter paths.
3. Offline packet flow and repository-provided query/AI scripts succeed against
   upgraded services.
4. Supported docs/config/deployments no longer instruct users to rely on
   Proto/gRPC tooling.

**Rationale**: This feature changes the repository’s contract technology and
multiple service surfaces at once. Fast unit tests alone are not enough; we
need evidence that the maintained workflows and operational assets moved
together.

**Alternatives considered**:

- **Rely on `go test ./...` only**: Rejected because it does not prove runtime
  scripts, generated artifacts, or supported deployment docs.
- **Treat docs/deployments as follow-up cleanup**: Rejected because the spec
  explicitly defines those as part of the supported cutover surface.

## Phase 1 Working Inventory

### Touched packages

- `internal/probe`, `pkg/pcap`, and `internal/engine/streamaggregator` for
  packet transport encoding/decoding and NATS consumption.
- `internal/engine/manager` to stop consuming generated Protobuf packet types.
- `internal/api`, `internal/query`, `cmd/ns-api/v1`, and `cmd/ns-api/v2` for
  query RPC and HTTP boundary migration.
- `internal/ai`, `internal/alerter`, and `cmd/ns-ai` for AI service and client
  migration.
- `scripts/query/v2` and `scripts/ask-ai` for repository-supported client
  workflows.

### Public and operational surfaces

- Legacy Proto/gRPC assets under `api/proto/v1/` and `api/gen/v1/`.
- New Thrift assets under `api/thrift/v1/` and `api/gen/thrift/v1/`.
- Runtime configuration in `configs/config.yaml` and `configs/config.yaml.example`.
- Documentation in `README.md`, `doc/build.md`, `doc/technology.md`,
  `doc/refactor/module-boundaries.md`, and relevant specs.
- Deployment assets in `deployments/kubernetes/` and
  `deployments/helm/go2netspectra/`.

### MVP scope

- Replace the maintained contract source of truth and generation flow.
- Remove Protobuf/gRPC from supported packet, query, AI, script, and ops paths.
- Preserve business capabilities while explicitly documenting breaking cutover
  behavior and unsupported mixed-version traffic.
