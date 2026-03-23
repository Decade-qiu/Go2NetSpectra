# Tasks: Repository-Wide Thrift Contract Migration

**Input**: Design documents from `/specs/002-thrift-rpc-migration/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Verification**: Verification tasks are REQUIRED because this feature changes
packet transport, contracts, config, query, AI, scripts, and deployment
behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Service entrypoints: `cmd/...`
- Core runtime logic: `internal/...`
- Public and derived contracts: `api/thrift/v1/`, `api/gen/thrift/v1/`, `api/proto/v1/`, `api/gen/v1/`
- Reusable libraries: `pkg/...`
- Runtime config and deployment: `configs/`, `deployments/`
- Fixtures and smoke assets: `test/`, `scripts/`

## Phase 1: Setup (Shared Context)

**Purpose**: Align the breaking cutover scope with the existing pipeline,
contracts, and operational surface

- [X] T001 Review `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/spec.md`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/plan.md`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/.specify/memory/constitution.md`; record touched binaries, packages, contracts, configs, scripts, and deployment assets in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/tasks.md`
- [X] T002 [P] Add or refresh contract-generation notes and fixture expectations in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md`
- [X] T003 [P] Inventory existing Proto/gRPC imports and runtime surface names across `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/configs`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/deployments`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish the new Thrift contract baseline and shared adapter seams

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T004 Create Thrift IDL source files in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/thrift/v1/traffic.thrift`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/thrift/v1/query.thrift`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/thrift/v1/ai.thrift`
- [X] T005 [P] Generate and verify Go bindings under `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/gen/thrift/v1/` and add regeneration guidance to `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/build.md`
- [X] T006 [P] Introduce shared transport/domain adapter seams in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/probe/packetcodec.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/engine/manager/manager.go`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/query/querier.go` so business logic no longer depends on generated Protobuf types
- [X] T007 Define Thrift RPC/session lifecycle, shutdown, expiry, and mixed-version failure behavior in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/contracts/thrift-service-contracts.md` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md`
- [X] T008 Sync foundational config and deployment naming targets for RPC cutover in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/config/config.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/configs/config.yaml`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/configs/config.yaml.example`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/contracts/cutover-sync-points.md`

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Cut Over Packet Transport Contracts (Priority: P1) 🎯 MVP

**Goal**: Replace Protobuf packet serialization on probe, offline analysis,
NATS transport, and manager ingress with Thrift while preserving packet field
semantics and worker/snapshot lifecycle behavior

**Independent Test**: Run focused packet tests plus the offline analyzer path,
then exercise the real-time probe/engine path to confirm Thrift packet payloads
can be encoded, published, consumed, decoded, and processed without any
Protocol Buffer decode path.

### Verification for User Story 1 ⚠️

- [X] T009 [P] [US1] Add or update packet codec tests in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/probe/packetcodec_test.go` for Thrift encode/decode, missing five-tuple handling, and legacy payload rejection
- [X] T010 [P] [US1] Add or update manager and stream-aggregator tests in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/engine/manager/manager_test.go` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/engine/streamaggregator/stream_aggregator_test.go` to validate Thrift-backed ingress and shutdown behavior
- [X] T011 [P] [US1] Add packet-path smoke coverage and execution notes in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/test/`

### Implementation for User Story 1

- [X] T012 [P] [US1] Replace Protobuf packet serialization helpers with Thrift serializer/deserializer logic in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/probe/packetcodec.go`
- [X] T013 [P] [US1] Update probe publish/subscribe paths to use the new Thrift packet contract in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/probe/publisher.go` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/probe/subscriber.go`
- [X] T014 [P] [US1] Update offline packet readers and any shared packet-entry helpers to emit the new contract/domain flow in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/pkg/pcap/` and related packet-handling files under `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/probe/`
- [X] T015 [US1] Refactor NATS consumption and manager ingress to accept Thrift-decoded domain packets in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/engine/streamaggregator/stream_aggregator.go` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/engine/manager/manager.go`
- [X] T016 [US1] Update packet-path documentation and boundary notes in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/pkg/pcap/doc.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd/ns-probe/doc.go`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/refactor/module-boundaries.md`

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - Replace Query And AI Service RPC (Priority: P2)

**Goal**: Replace query-service and AI-service Protobuf/gRPC contracts with
Thrift RPC, decouple business logic from generated transport structs, and keep
repository-provided clients working through the new RPC surface

**Independent Test**: Start the upgraded query and AI services, run the
repository query and AI scripts through the Thrift client path, and confirm
health, task search, aggregate, trace, heavy-hitter, alert-analysis, and
prompt-session chunk retrieval all work without gRPC clients.

### Verification for User Story 2 ⚠️

- [X] T017 [P] [US2] Add or update query adapter tests in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/api/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/query/`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/api/http_server_test.go`
- [X] T018 [P] [US2] Add or update AI service and alerter client tests in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/ai/` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/alerter/`
- [X] T019 [P] [US2] Add or update client smoke steps for query and AI flows in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/query/v2/main.go`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/ask-ai/main.go`

### Implementation for User Story 2

- [X] T020 [P] [US2] Introduce internal query request/response DTOs and adapt ClickHouse query code away from generated Protobuf types in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/query/querier.go`
- [X] T021 [P] [US2] Replace the query gRPC server with a Thrift RPC server and update supported HTTP handlers to avoid `protojson` in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/api/grpc_server.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/api/http_server.go`, and related files under `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/api/`
- [X] T022 [P] [US2] Replace the AI gRPC server with a Thrift RPC server, including prompt-analysis session management and chunk retrieval, in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/ai/server.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/ai/common_analyzer.go`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/ai/`
- [X] T023 [P] [US2] Update AI and query clients to use Thrift RPC in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/alerter/alerter.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/query/v2/main.go`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/ask-ai/main.go`
- [X] T024 [US2] Update service entrypoints and supported command wiring for the new RPC servers in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd/ns-api/v2/main.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd/ns-api/v1/main.go`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd/ns-ai/main.go`
- [X] T025 [US2] Remove maintained-path dependencies on legacy Proto/gRPC generated bindings by switching imports and regeneration references across `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/gen/thrift/v1/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd/`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/`

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - Retire Legacy Protocol Operations Surface (Priority: P3)

**Goal**: Remove Proto/gRPC from the supported build, config, script, and
deployment surface so operators and maintainers only see the Thrift workflow

**Independent Test**: Follow the supported docs and config/deployment assets to
regenerate contracts, start services, and run the repository-provided clients
without any Protobuf/gRPC toolchain or runtime instructions.

### Verification for User Story 3 ⚠️

- [X] T026 [P] [US3] Add or update config/deployment validation notes in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/contracts/cutover-sync-points.md`
- [X] T027 [P] [US3] Add or update build/documentation verification steps in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/README.md`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/build.md`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/technology.md`

### Implementation for User Story 3

- [X] T028 [P] [US3] Rename runtime config fields and load paths from gRPC-specific names to RPC-neutral names in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/config/config.go`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/configs/config.yaml`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/configs/config.yaml.example`
- [X] T029 [P] [US3] Update Kubernetes and Helm manifests to expose Thrift/RPC runtime surfaces in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/deployments/kubernetes/` and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/deployments/helm/go2netspectra/`
- [X] T030 [P] [US3] Replace supported build and architecture documentation for Proto/gRPC with Thrift guidance in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/README.md`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/build.md`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/technology.md`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/doc/refactor/module-boundaries.md`
- [X] T031 [US3] Retire or explicitly mark legacy Proto/gRPC assets as unsupported in maintained workflows across `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/proto/v1/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/api/gen/v1/`, and affected repository docs
- [X] T032 [US3] Validate cross-story operational behavior and explicit legacy failure messaging in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/`, and supported deployment assets

**Checkpoint**: All user stories should now be independently functional

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, cleanup, and delivery synchronization across all stories

- [X] T033 [P] Run `gofmt` and `goimports` on touched Go files under `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/internal/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/cmd/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/pkg/`
- [X] T034 Execute targeted Go tests and repository-wide verification commands from `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/quickstart.md`
- [X] T035 [P] Validate supported client, NATS packet, and deployment workflows from `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/scripts/`, `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/deployments/`, and `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/configs/`
- [X] T036 Update any remaining spec, plan, or review notes in `/Users/qzj/Desktop/Development/Traffic-Monitor/Go2NetSpectra/specs/002-thrift-rpc-migration/` and capture skipped validation or follow-up debt explicitly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel if staffed
  - Or sequentially in priority order (P1 -> P2 -> P3)
- **Polish (Phase 6)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2), but benefits from the shared contract baseline and adapter seams established for US1
- **User Story 3 (P3)**: Can start after Foundational (Phase 2), but should land after US1/US2 runtime behavior is stable so docs and deployments reflect delivered behavior

### Within Each User Story

- Verification tasks should start before or alongside implementation
- Contract and adapter updates come before generated import rewiring and service/client wiring
- Runtime implementation completes before docs/deployment sync for that story
- Story-specific smoke validation completes before moving to the next story if working sequentially

### Parallel Opportunities

- `T002`, `T003`, and `T005` can run in parallel once scope is fixed
- `T006` and `T008` can proceed in parallel after the Thrift IDL shape is stable
- Within US1, `T009`, `T010`, `T011`, `T012`, `T013`, and `T014` can be split across different files/packages
- Within US2, query work (`T017`, `T020`, `T021`) and AI work (`T018`, `T022`, `T023`) can proceed in parallel
- Within US3, config/deployment work (`T028`, `T029`) and doc cleanup (`T027`, `T030`) can proceed in parallel

---

## Parallel Example: User Story 1

```bash
# Launch verification for User Story 1 together:
Task: "Add Thrift packet codec tests in internal/probe/packetcodec_test.go"
Task: "Add manager and stream-aggregator packet ingress tests in internal/engine/manager/manager_test.go and internal/engine/streamaggregator/stream_aggregator_test.go"

# Launch independent implementation work together:
Task: "Replace packet serialization in internal/probe/packetcodec.go"
Task: "Update probe publish/subscribe flow in internal/probe/publisher.go and internal/probe/subscriber.go"
Task: "Refactor manager ingress and NATS consumer flow in internal/engine/streamaggregator/stream_aggregator.go and internal/engine/manager/manager.go"
```

---

## Parallel Example: User Story 2

```bash
# Launch query and AI verification together:
Task: "Add query adapter tests in internal/api/ and internal/query/"
Task: "Add AI service and alerter client tests in internal/ai/ and internal/alerter/"

# Launch independent implementation work together:
Task: "Move internal/query/querier.go onto internal DTOs"
Task: "Replace query server wiring in internal/api/"
Task: "Replace AI server and session handling in internal/ai/"
Task: "Update scripts/query/v2/main.go and scripts/ask-ai/main.go to use Thrift clients"
```

---

## Parallel Example: User Story 3

```bash
# Launch operational cleanup together:
Task: "Rename rpc config fields in internal/config/config.go and configs/"
Task: "Update Kubernetes and Helm assets in deployments/"
Task: "Replace Proto/gRPC guidance in README.md and doc/"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run packet-path tests and offline/real-time packet smokes
5. Demo the Thrift packet cutover before touching service RPC

### Incremental Delivery

1. Complete Setup + Foundational -> Foundation ready
2. Add User Story 1 -> Test independently -> Stabilize packet path
3. Add User Story 2 -> Test independently -> Stabilize query/AI RPC path
4. Add User Story 3 -> Test independently -> Finalize operational surface
5. Finish with Phase 6 polish and explicit evidence capture

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1 packet transport path
   - Developer B: User Story 2 query RPC path
   - Developer C: User Story 2 AI/session path plus User Story 3 operational sync
3. Rejoin for final polish, integrated smoke runs, and release notes

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable after the foundational phase
- Regenerate `api/gen/thrift/v1/` whenever `.thrift` files change
- Keep `api/proto/v1/` and `api/gen/v1/` as legacy history only after the cutover, not as maintained supported paths
- Sync `configs/`, `deployments/`, `README.md`, and `doc/` whenever runtime names, startup steps, or contract tooling change
