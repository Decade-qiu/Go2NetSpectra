# Tasks: Repository-Wide Go Refactor Program

**Input**: Design documents from `/specs/001-repo-go-refactor/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Verification**: Verification tasks are REQUIRED because this refactor touches
packet parsing, aggregation, contracts, config, query, alerting, AI, and
deployment behavior.

**Organization**: Tasks are grouped by user story so each phase can be
implemented, reviewed, and validated as an independent increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Task can run in parallel with other tasks when it touches different files and has no unfinished dependency.
- **[Story]**: User story label for story-specific phases (`[US1]`, `[US2]`, `[US3]`).
- Every task includes exact repository file paths.

## Path Conventions

- Service entrypoints: `cmd/...`
- Core runtime logic: `internal/...`
- Public and derived contracts: `api/proto/v1/`, `api/gen/v1/`
- Reusable libraries: `pkg/...`
- Runtime config and deployment: `configs/`, `deployments/`
- Fixtures, smoke assets, and planning evidence: `test/`, `scripts/`, `specs/001-repo-go-refactor/`

## Phase 1: Setup (Shared Context)

**Purpose**: Lock the baseline, working notes, and compatibility evidence before structural changes begin.

- [X] T001 Review `specs/001-repo-go-refactor/spec.md`, `specs/001-repo-go-refactor/plan.md`, and `.specify/memory/constitution.md`; capture touched packages, public surfaces, and phase scope in `specs/001-repo-go-refactor/research.md`
- [X] T002 [P] Create the repository ownership map and target package additions in `doc/refactor/module-boundaries.md`
- [X] T003 [P] Capture baseline validation steps, hot-path evidence tables, and exit-evidence placeholders in `specs/001-repo-go-refactor/quickstart.md` and `specs/001-repo-go-refactor/contracts/refactor-exit-criteria.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared seams, validation harnesses, and contract-sync rules that all user stories depend on.

**⚠️ CRITICAL**: No user story work should start before this phase is complete.

- [X] T004 Create the shared packet conversion seam in `internal/probe/packetcodec.go` and `internal/probe/packetcodec_test.go`
- [X] T005 [P] Add package ownership docs in `internal/protocol/doc.go`, `internal/probe/doc.go`, `internal/query/doc.go`, `internal/engine/manager/doc.go`, and `internal/factory/doc.go`
- [X] T006 [P] Create repository validation entrypoints in `scripts/lint.sh`, `scripts/build.sh`, and `doc/build.md`
- [X] T007 Define contract/config/deployment and plugin-registration sync rules in `README.md`, `configs/config.yaml`, `doc/refactor/module-boundaries.md`, `deployments/docker-compose/docker-compose.yml`, and `deployments/helm/go2netspectra/values.yaml`
- [X] T008 [P] Create lifecycle and plugin-registration regression harnesses in `internal/engine/manager/manager_test.go`, `internal/probe/persistent/worker_test.go`, and `internal/factory/task_factory_test.go`

**Checkpoint**: Shared seams, docs, and validation hooks are ready for story-level implementation.

---

## Phase 3: User Story 1 - Clarify Ownership Boundaries (Priority: P1) 🎯 MVP

**Goal**: Make runtime ownership boundaries explicit so maintainers can locate and change the correct layer without cross-directory guesswork.

**Independent Test**: Follow the updated ownership map and trace one representative packet-processing path from ingress to query/alert usage, confirming the responsible module and entrypoint can be identified quickly and consistently.

### Verification for User Story 1

- [X] T009 [P] [US1] Add ownership regression coverage for shared packet conversion and manager routing in `internal/probe/packetcodec_test.go` and `internal/engine/manager/manager_test.go`
- [X] T010 [P] [US1] Add offline, live-probe, query, alerting, and service-boundary smoke instructions in `doc/refactor/module-boundaries.md` and `specs/001-repo-go-refactor/quickstart.md`

### Implementation for User Story 1

- [X] T011 [P] [US1] Extract shared protobuf packet conversion from `internal/probe/publisher.go`, `internal/probe/subscriber.go`, and `pkg/pcap/reader.go` into `internal/probe/packetcodec.go`
- [X] T012 [P] [US1] Move `ns-api` server assembly out of `cmd/ns-api/v1/main.go` and `cmd/ns-api/v2/main.go` into `internal/api/http_server.go` and `internal/api/grpc_server.go`
- [X] T013 [P] [US1] Move `ns-ai`, `ns-engine`, and offline analyzer bootstrapping out of `cmd/ns-ai/main.go`, `cmd/ns-engine/main.go`, and `cmd/pcap-analyzer/main.go` into the long-term runtime homes `internal/ai/server.go` and `internal/engine/app/runner.go`
- [X] T014 [US1] Update ownership references and entrypoint guidance in `README.md`, `doc/build.md`, and `doc/refactor/module-boundaries.md`

**Checkpoint**: The repository has a documented and testable ownership model for the main runtime path.

---

## Phase 4: User Story 2 - Standardize Repository Conventions (Priority: P2)

**Goal**: Make naming, comments, errors, context use, imports, and CLI messaging predictable across the maintained Go codebase.

**Independent Test**: Sample representative files from entrypoints, shared runtime packages, and hotspot modules; reviewers should be able to apply one consistent rule set without relying on per-directory historical habits.

### Verification for User Story 2

- [X] T015 [P] [US2] Add convention regression tests for config loading and query behavior in `internal/config/config_test.go` and `internal/query/querier_test.go`
- [X] T016 [P] [US2] Add formatting and convention checks in `scripts/lint.sh` and `AGENTS.md`

### Implementation for User Story 2

- [X] T017 [P] [US2] Standardize package/file naming and package docs for `internal/model/Packet.go` -> `internal/model/packet.go`, `internal/model/Task.go` -> `internal/model/task.go`, `internal/engine/impl/sketch/statistic/CountMin.go` -> `internal/engine/impl/sketch/statistic/count_min.go`, and `internal/engine/impl/sketch/statistic/SuperSpread.go` -> `internal/engine/impl/sketch/statistic/super_spread.go`
- [X] T018 [P] [US2] Normalize comments, errors, and context usage in `internal/config/config.go`, `internal/query/querier.go`, `internal/ai/alerter_analyzer.go`, and `internal/ai/common_analyzer.go`
- [X] T019 [P] [US2] Normalize lifecycle and error-handling conventions in `internal/alerter/alerter.go`, `internal/engine/manager/manager.go`, `internal/engine/streamaggregator/stream_aggregator.go`, and `internal/probe/persistent/worker.go`
- [X] T020 [US2] Align CLI/script messaging and developer guidance in `cmd/ns-probe/main.go`, `scripts/query/v1/main.go`, `scripts/query/v2/main.go`, `scripts/ask-ai/main.go`, and `doc/go-codex-style.md`

**Checkpoint**: The maintained codebase presents a single, repository-level Go convention set.

---

## Phase 5: User Story 3 - Improve Hot-Path Efficiency Safely (Priority: P3)

**Goal**: Improve or preserve prioritized hot-path performance while keeping correctness, concurrency safety, and shutdown behavior intact.

**Independent Test**: Compare baseline and post-change evidence for parser, manager, probe, and exact/sketch hotspots; confirm required runtime workflows still pass and no unacceptable regression remains.

### Verification for User Story 3

- [X] T021 [P] [US3] Expand benchmark coverage and record baseline thresholds for parser, exact, and sketch hotspots in `internal/engine/impl/benchmark/perf_test.go`, `scripts/hash/hash_bench_test.go`, and `specs/001-repo-go-refactor/contracts/refactor-exit-criteria.md`
- [X] T022 [P] [US3] Add fixture-driven non-regression coverage in `internal/protocol/parser_test.go`, `internal/engine/impl/sketch/cm_test.go`, and `internal/engine/impl/sketch/ss_test.go`

### Implementation for User Story 3

- [X] T023 [P] [US3] Reduce duplicate packet marshaling and conversion allocations in `internal/probe/publisher.go`, `internal/probe/subscriber.go`, `pkg/pcap/reader.go`, and `internal/protocol/parser.go`
- [X] T024 [US3] Optimize manager and worker lifecycle coordination in `internal/engine/manager/manager.go`, `internal/engine/streamaggregator/stream_aggregator.go`, and `internal/probe/persistent/worker.go`
- [X] T025 [US3] Optimize exact/sketch hot paths in `internal/engine/impl/exact/task.go`, `internal/engine/impl/sketch/task.go`, `internal/engine/impl/sketch/statistic/count_min.go`, and `internal/engine/impl/sketch/statistic/super_spread.go` after `T017` completes
- [X] T026 [US3] Optimize query/write non-regression paths in `internal/query/querier.go`, `internal/engine/impl/exact/writer_clickhouse.go`, and `internal/engine/impl/sketch/writer_clickhouse.go`

**Checkpoint**: Priority hotspots have benchmark-backed improvements or accepted non-regression evidence.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Close the loop on repository-wide validation, docs, and deferred debt.

- [X] T027 [P] Run `gofmt` and `goimports` on touched files in `cmd/`, `internal/`, `pkg/`, and `scripts/`
- [X] T028 Execute repository regression suites referenced in `specs/001-repo-go-refactor/quickstart.md`
- [ ] T029 [P] Validate offline, live-probe, query, alerting, and AI smoke workflows from `specs/001-repo-go-refactor/quickstart.md`
- [X] T030 [P] Sync final contract/config/deployment docs in `README.md`, `doc/build.md`, `deployments/docker-compose/docker-compose.yml`, and `deployments/helm/go2netspectra/values.yaml`
- [X] T031 Capture deferred debt and phase outcomes in `specs/001-repo-go-refactor/contracts/refactor-exit-criteria.md` and `doc/refactor/module-boundaries.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** — starts immediately.
- **Phase 2: Foundational** — depends on setup and blocks all story work.
- **Phase 3: US1** — starts after foundational work and defines the MVP.
- **Phase 4: US2** — starts after foundational work; may reuse boundary outputs from US1.
- **Phase 5: US3** — starts after foundational work; benefits from US1 ownership cleanup and US2 convention normalization.
- **Phase 6: Polish** — starts after the desired user stories are complete.

### User Story Dependencies

- **US1 (P1)**: Depends only on foundational work.
- **US2 (P2)**: Depends on foundational work; integrates with US1 boundary outputs but remains independently reviewable.
- **US3 (P3)**: Depends on foundational work and on the naming/lifecycle normalization from `T017`-`T019` before starting `T024`-`T026`.

### Within Each User Story

- Verification tasks come before or alongside structural edits.
- Shared seams and helpers come before entrypoint rewiring.
- Runtime documentation updates finish the story so ownership and compatibility remain visible.
- Story completion requires both implementation and the story’s independent validation path.

### Parallel Opportunities

- Setup: `T002` and `T003`
- Foundational: `T005`, `T006`, and `T008`
- US1: `T009`, `T010`, `T011`, `T012`, and `T013`
- US2: `T015`, `T016`, `T017`, `T018`, and `T019`
- US3: `T021`, `T022`, and `T023`
- Polish: `T027`, `T029`, and `T030`

---

## Parallel Example: User Story 1

```bash
Task: "T009 [US1] Add ownership regression coverage in internal/probe/packetcodec_test.go and internal/engine/manager/manager_test.go"
Task: "T010 [US1] Add offline, live-probe, query, alerting, and service-boundary smoke instructions in doc/refactor/module-boundaries.md and specs/001-repo-go-refactor/quickstart.md"
Task: "T011 [US1] Extract shared packet conversion into internal/probe/packetcodec.go"
Task: "T012 [US1] Move ns-api server assembly into internal/api/http_server.go and internal/api/grpc_server.go"
Task: "T013 [US1] Move runtime bootstrapping into internal/ai/server.go and internal/engine/app/runner.go"
```

## Parallel Example: User Story 2

```bash
Task: "T015 [US2] Add convention regression tests in internal/config/config_test.go and internal/query/querier_test.go"
Task: "T016 [US2] Add formatting and convention checks in scripts/lint.sh and AGENTS.md"
Task: "T017 [US2] Standardize naming/docs and rename Packet.go/Task.go and sketch statistic files"
Task: "T018 [US2] Normalize comments/errors/context usage in internal/config/config.go, internal/query/querier.go, and internal/ai/*.go"
Task: "T019 [US2] Normalize lifecycle/error handling in manager, streamaggregator, alerter, and persistent worker files"
```

## Parallel Example: User Story 3

```bash
Task: "T021 [US3] Expand benchmarks and record baseline thresholds in internal/engine/impl/benchmark/perf_test.go, scripts/hash/hash_bench_test.go, and specs/001-repo-go-refactor/contracts/refactor-exit-criteria.md"
Task: "T022 [US3] Add fixture-driven non-regression coverage in parser and sketch tests"
Task: "T023 [US3] Reduce conversion allocations in publisher, subscriber, pcap reader, and parser files"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Setup.
2. Complete Foundational work.
3. Deliver User Story 1.
4. Run the US1 independent test path from `doc/refactor/module-boundaries.md` and `specs/001-repo-go-refactor/quickstart.md`.
5. Stop for review before broader convention and performance work.

### Incremental Delivery

1. Setup + Foundational -> shared seams and validation ready.
2. US1 -> repository ownership becomes navigable and reviewable.
3. US2 -> conventions become consistent across the maintained codebase.
4. US3 -> hotspot performance work lands on top of a cleaner structure.
5. Polish -> repository-wide regression, smoke, and docs sync finish the program.

### Parallel Team Strategy

1. One owner handles foundational seams and validation (`T004`-`T008`).
2. After foundational completion:
   - Developer A owns US1 boundary extraction and entrypoint rewiring.
   - Developer B owns US2 convention cleanup and lint/doc alignment, including the overlapping files in `T017`-`T019`.
   - Developer C owns US3 baseline capture and isolated hot-path work (`T021`-`T023`), then starts `T024`-`T026` only after `T017`-`T019` land.
3. Polish tasks close shared docs, validation, and deferred-debt tracking.

---

## Notes

- Total tasks: 31
- User story tasks: US1 = 6, US2 = 6, US3 = 6
- Setup/foundational/polish tasks: 13
- Regenerate `api/gen/v1/` whenever `.proto` files change during the refactor.
- Keep `cmd/` focused on process wiring; shared runtime logic belongs under `internal/`.
- Record benchmark baselines and non-regression thresholds in `specs/001-repo-go-refactor/contracts/refactor-exit-criteria.md`.
- Stop at each story checkpoint and record any deferred debt explicitly.
