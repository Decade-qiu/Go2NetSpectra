# Implementation Plan: Repository-Wide Thrift Contract Migration

**Branch**: `002-thrift-rpc-migration` | **Date**: 2026-03-23 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-thrift-rpc-migration/spec.md`

**Note**: This plan converts the approved breaking cutover from Protobuf/gRPC to
Thrift/Thrift RPC into a staged design that preserves current business
capabilities while changing the contract technology, generated artifacts, and
operational surface.

## Summary

Replace the repository-wide Protocol Buffer/gRPC contract layer with a new
Thrift source of truth, covering packet transport on NATS, query service RPC,
AI service RPC, and all repository-supported scripts, docs, configs, and
deployment assets. The design keeps the existing capture -> transport ->
manager -> task -> writer/query/alert/AI layering, but moves runtime/business
logic away from generated wire structs, introduces Thrift-generated bindings
under a new contract path, and models AI prompt streaming as a session-based
chunk-pull workflow because Thrift does not provide a native gRPC-style
server-streaming primitive.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**: gopacket, Apache Thrift compiler and Go library,
NATS, ClickHouse, YAML configuration, go-openai, gorilla/mux
**Storage**: ClickHouse, optional gob/text/pcap outputs, NATS as packet
transport, YAML configuration, generated Thrift artifacts under
`api/gen/thrift/v1`, and fixture data under `test/`
**Testing**: `go test ./...`, focused package tests for `internal/probe`,
`internal/engine/manager`, `internal/engine/streamaggregator`, `internal/api`,
`internal/query`, `internal/ai`, and `internal/alerter`, plus pcap-fixture
validation, client-script smoke tests, contract generation checks, and hot-path
benchmark comparison when packet processing cost changes
**Target Platform**: Linux/macOS development plus containerized Linux for
Docker, Helm, and Kubernetes deployments
**Project Type**: Distributed Go services, CLI tools, generated contract
artifacts, and deployment assets
**Performance Goals**: Preserve packet-processing throughput and query/AI
usability after the cutover, avoid unbounded allocation growth on NATS packet
encoding/decoding, and document/verify goroutine cleanup for prompt-analysis
sessions
**Constraints**: Breaking cutover only; no mixed Protobuf/gRPC and Thrift
interoperability; preserve `cmd/`/`internal/`/`pkg/` boundaries, plugin
registration, read-only snapshots, graceful shutdown, and environment-driven
secrets; keep supported HTTP endpoints free of Protobuf-only DTO coupling
**Scale/Scope**: Multi-service contract migration across IDL, generated Go
artifacts, packet ingestion, query and AI RPCs, repository scripts, config
schema, documentation, and deployment manifests

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Pipeline layering is preserved. The plan keeps the same
      capture/parse -> transport contract/NATS -> manager -> task group ->
      writer/query/alert/AI flow, changing the wire technology but not adding a
      parallel runtime path.
- [x] Plugin registration remains authoritative. Exact/sketch task creation,
      `model.Task` / `model.Writer`, `factory.RegisterAggregator`, config-driven
      registration, and manager blank imports are unchanged by the migration.
- [x] Contract synchronization is explicit. The cutover enumerates old and new
      contract locations, generated artifacts, handlers/clients, config keys,
      scripts, and deployment assets that must move in lockstep.
- [x] Concurrency design is documented. Packet workers keep existing ownership,
      and new prompt-analysis sessions define explicit creation, chunk-drain,
      cancellation, expiry, and shutdown rules.
- [x] Verification is concrete. The plan names focused Go tests, pcap and
      client smokes, generation checks, and deployment/runtime validation for
      the touched paths.

## Project Structure

### Documentation (this feature)

```text
specs/002-thrift-rpc-migration/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── cutover-sync-points.md
│   └── thrift-service-contracts.md
└── tasks.md
```

### Source Code (repository root)

```text
api/
├── proto/v1/           # legacy source of truth to retire at cutover
├── gen/v1/             # legacy generated protobuf/go-grpc outputs to retire
├── thrift/v1/          # new Thrift IDL source of truth
└── gen/thrift/v1/      # generated Go bindings for Thrift
cmd/
├── ns-ai/
├── ns-api/v1/
├── ns-api/v2/
├── ns-engine/
├── ns-probe/
└── pcap-analyzer/
internal/
├── ai/
├── alerter/
├── config/
├── engine/
│   ├── impl/exact/
│   ├── impl/sketch/
│   ├── manager/
│   └── streamaggregator/
├── factory/
├── model/
├── notification/
├── probe/
├── protocol/
└── query/
pkg/
└── pcap/
scripts/
deployments/
configs/
doc/
README.md
test/
```

**Structure Decision**: Add a new Thrift contract root under `api/thrift/v1`
and generated Go bindings under `api/gen/thrift/v1`, retire
`api/proto/v1`/`api/gen/v1` from supported workflows at cutover, and keep
runtime/business logic inside existing `internal/` packages. The migration may
add internal adapter types near `internal/probe`, `internal/query`, and
`internal/ai`, but it does not move reusable logic into `cmd/` or introduce a
new top-level runtime boundary.

## Phase 0: Research Summary

Research resolves the core contract and migration design decisions:

1. Establish `api/thrift/v1` as the only supported contract source of truth and
   keep generated Go code derived under `api/gen/thrift/v1`.
2. Use Thrift binary protocol consistently, with serializer/deserializer-based
   payload handling for NATS packets and buffered TCP transports for RPC.
3. Move manager/query/HTTP flows onto internal request-response/domain models
   so generated Thrift structs stop leaking into business logic.
4. Replace the AI streaming RPC with a Thrift-native session/chunk pull model
   that preserves incremental output without keeping gRPC.
5. Rename protocol-specific runtime surfaces from `grpc_*`/`grpc` to neutral
   `rpc_*`/`rpc` names where they describe endpoint roles rather than contract
   file locations.
6. Validate the cutover with a ladder covering generation, focused tests,
   offline packet flow, query/AI client smokes, and supported ops assets.

See [research.md](./research.md) for the full decision log.

## Phase 1: Design

### Work Packages

1. **Contract Source And Generation**
   - Create Thrift IDL files for packet transport, query service, and AI service.
   - Define the generated output layout and regeneration expectations.
   - Record which legacy Proto/gRPC files are retired versus kept only as
     repository history.

2. **Packet Pipeline Migration**
   - Replace Protobuf packet marshaling/unmarshaling in `internal/probe`,
     `pkg/pcap`, `internal/engine/streamaggregator`, and
     `internal/engine/manager`.
   - Shift manager ingress from generated wire structs to domain packet models
     so the worker pool no longer depends on a specific serialization library.
   - Preserve snapshot, reset, and shutdown behavior.

3. **Query And HTTP Boundary Migration**
   - Replace gRPC query server/client code with Thrift RPC handlers and clients.
   - Introduce internal query request/response models so `internal/query` and
     HTTP handlers do not depend on generated transport code.
   - Preserve supported health, task-search, aggregate, trace, and
     heavy-hitter behaviors.

4. **AI RPC And Prompt Session Migration**
   - Replace AI gRPC server/client code with Thrift RPC.
   - Keep unary traffic analysis as a direct request/response operation.
   - Convert prompt streaming into explicit prompt-analysis sessions with chunk
     reads, cancellation, expiry, and graceful shutdown semantics.

5. **Operations Surface Cutover**
   - Rename protocol-specific config keys and deployment port names.
   - Update scripts, docs, Helm/Kubernetes assets, build instructions, and any
     runtime smoke steps to reference Thrift instead of Proto/gRPC.
   - Make mixed-version failures explicit and documented.

### Data Model

The design centers on contract artifacts, packet transport messages, query
operations, flow and heavy-hitter result models, and AI prompt-analysis
sessions. See [data-model.md](./data-model.md).

### Contracts

The feature exposes explicit Thrift contract surfaces and cutover sync rules:

- [thrift-service-contracts.md](./contracts/thrift-service-contracts.md)
  documents the new packet, query, and AI contracts, including the replacement
  for server-streaming AI prompts.
- [cutover-sync-points.md](./contracts/cutover-sync-points.md) lists the code,
  config, script, doc, and deployment surfaces that must be updated together.

### Quickstart Validation

The validation workflow covers Thrift code generation, focused tests, packet
pipeline smoke, query/AI client smoke, and supported-asset checks. See
[quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- [x] The design preserves the single transport-contract layer and keeps the
      rest of the runtime topology unchanged.
- [x] Aggregator and writer registration still flow only through existing
      config/factory/task seams.
- [x] Contract/config/deployment synchronization is modeled explicitly through
      the cutover sync matrix and quickstart validation.
- [x] New concurrency is bounded to prompt-analysis session management and is
      documented with ownership, cancellation, expiry, and shutdown rules.
- [x] Verification covers generated artifacts, package tests, packet fixtures,
      service/client smokes, and operational surface checks.

## Implementation Phases

### Phase 1: Establish Thrift Contract Baseline

- Add `api/thrift/v1` IDL files for packet, query, and AI contracts.
- Add generated Go outputs under `api/gen/thrift/v1` and document regeneration.
- Keep legacy Proto/gRPC files as retired artifacts until final code removal,
  but stop treating them as supported sources of truth.

### Phase 2: Decouple Packet Transport From Generated Wire Types

- Replace Protobuf codec usage in publishers, subscribers, packet readers, and
  NATS consumers with Thrift codec helpers.
- Change manager ingress to consume domain packet models instead of generated
  transport structs.
- Preserve existing worker-pool, snapshot, reset, and final-flush semantics.

### Phase 3: Replace Query RPC And HTTP Adapters

- Add Thrift query server/client adapters and remove gRPC query dependencies.
- Move `internal/query` onto internal DTOs and map JSON/HTTP requests directly
  to those DTOs rather than generated contract structs.
- Update repository query scripts and supported HTTP endpoints.

### Phase 4: Replace AI RPC And Streaming Behavior

- Add Thrift AI server/client adapters and remove gRPC AI dependencies.
- Migrate unary alert-analysis requests directly.
- Implement session-based prompt chunk retrieval with explicit cancellation,
  expiry, and shutdown cleanup for the streaming use case.

### Phase 5: Cut Over Operations Surface And Retire Proto/gRPC Support

- Rename config and deployment surfaces away from `grpc_*` naming where those
  names describe active runtime roles.
- Update README, build docs, technology docs, deployment manifests, Helm
  values, and smoke instructions.
- Remove supported runtime dependencies on `google.golang.org/grpc`,
  `google.golang.org/protobuf`, and `protojson` from maintained paths.

## Complexity Tracking

No constitution violations are currently required. The plan preserves the same
layered runtime topology and plugin model; the additional complexity is limited
to the AI prompt session workflow, which is necessary to replace a gRPC
streaming contract with a Thrift-native equivalent.
