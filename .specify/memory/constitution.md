<!--
Sync Impact Report
- Version change: template -> 1.0.0
- Modified principles:
  - placeholder principle 1 -> I. Pipeline-First Layering
  - placeholder principle 2 -> II. Config-Driven Plugin Extension
  - placeholder principle 3 -> III. Contract And Config Synchronization
  - placeholder principle 4 -> IV. Concurrency, Snapshot Consistency, And Graceful Shutdown
  - placeholder principle 5 -> V. Verification On Real Traffic Paths
- Added sections:
  - Architecture Boundaries
  - Delivery Workflow
- Removed sections:
  - None
- Templates requiring updates:
  - ✅ .specify/templates/plan-template.md
  - ✅ .specify/templates/spec-template.md
  - ✅ .specify/templates/tasks-template.md
  - ✅ .specify/templates/agent-file-template.md
  - ⚠ .specify/templates/commands/*.md (directory missing; no action taken)
- Follow-up TODOs:
  - None
-->
# Go2NetSpectra Constitution

## Core Principles

### I. Pipeline-First Layering
- All changes MUST preserve the current pipeline boundaries: capture and parsing
  (`cmd/ns-probe`, `pkg/pcap`, `internal/protocol`) -> transport contracts
  (`api/proto/v1`, `api/gen/v1`, NATS) -> orchestration
  (`internal/engine/manager`, `internal/engine/streamaggregator`) -> execution
  (`internal/engine/impl/*`) -> persistence, query, alerting, and AI
  (`internal/query`, `internal/alerter`, `internal/ai`, `cmd/ns-api`, `cmd/ns-ai`).
- Cross-layer shortcuts MUST NOT bypass protobuf contracts, the manager/task
  seam, or the configured transport path only to save code or hops.
- New runtime behavior MUST fit an existing layer boundary or justify a new one
  in the implementation plan before code begins.

Rationale: the project’s value comes from reusing the same traffic pipeline for
offline analysis, real-time streaming, query, alerting, and AI enrichment.

### II. Config-Driven Plugin Extension
- New aggregators and writers MUST be driven from `configs/config.yaml`,
  created through `internal/factory`, and activated with both
  `factory.RegisterAggregator` and the required blank import in
  `internal/engine/manager`.
- `model.Task` and `model.Writer` are the default extension seams. New behavior
  MUST prefer extending those seams over introducing parallel frameworks or
  sidecar registries.
- Any change to task names, flow fields, writer outputs, or storage tables MUST
  document the config, storage, and client impact in plan and task artifacts.

Rationale: exact and sketch analysis are intentionally pluggable, and that
pluggability depends on configuration and registration discipline.

### III. Contract And Config Synchronization
- Files under `api/proto/v1/` and `configs/config.yaml` are authoritative public
  contracts. Any change MUST regenerate `api/gen/v1/`, update all affected
  servers and clients, and sync Docker, Helm, Kubernetes, or environment docs
  when runtime inputs change.
- Service addresses, credentials, and deployment-specific values MUST remain
  environment-driven. Secrets MUST NOT be hard-coded in Go code or committed
  developer/runtime instructions.
- Query, alerting, and AI interface changes MUST state compatibility
  expectations for existing data, clients, and deployment procedures.

Rationale: this repository ships multiple binaries and deployment modes, so
unsynchronized contracts are a primary source of regressions.

### IV. Concurrency, Snapshot Consistency, And Graceful Shutdown
- Every new goroutine, channel, ticker, snapshot loop, or reset loop MUST have
  explicit ownership, stop conditions, and shutdown behavior.
- `Snapshot()` implementations MUST remain read-only views of current state, and
  measurement-period reset behavior MUST stay coordinated by manager-level
  orchestration unless an alternative design is documented and verified.
- Hot-path packet processing MUST avoid hidden shared-state coupling, data
  races, and unbounded allocation growth. Concurrency safety is part of feature
  completeness, not a later optimization pass.

Rationale: throughput matters here, but correctness under sustained concurrent
traffic matters more.

### V. Verification On Real Traffic Paths
- Changes to parsing, aggregation, query, alerting, AI workflows, or deployment
  behavior MUST include executable verification on the closest relevant path:
  focused `go test`, pcap-fixture validation, client-script exercise, benchmark
  evidence, or deployment smoke validation.
- Performance-sensitive changes under `internal/protocol`, `internal/probe`, or
  `internal/engine/impl/*` MUST capture impact with benchmarks or realistic
  traffic fixtures when the behavior or complexity materially changes.
- A feature is not complete until affected docs, scripts, and operational steps
  are updated enough for another developer to run the path end to end.

Rationale: Go2NetSpectra is a traffic-monitoring product; correctness without
runtime proof is incomplete.

## Architecture Boundaries

- `cmd/` contains process wiring, startup, and shutdown orchestration only.
  Reusable business logic belongs in `internal/` or `pkg/`.
- `pkg/` is reserved for code intentionally reusable outside a single binary.
  Engine internals, query routing, alerting, AI integration, and probe runtime
  logic stay in `internal/`.
- Generated protobuf files under `api/gen/v1/` are derived artifacts and MUST
  be regenerated, not hand-maintained.
- Exact and sketch implementations MAY diverge internally, but shared service
  surfaces MUST document which backend is authoritative for each query or
  storage path.
- Deployment assets under `deployments/` and defaults under `configs/` are part
  of the product surface and MUST stay runnable alongside code changes.

## Delivery Workflow

- Every implementation plan MUST name the exact binaries, packages, contracts,
  configs, scripts, and deployment assets touched by the feature.
- Every task list MUST include regeneration or sync work when touching
  `.proto`, config, deployment manifests, task registration, or client scripts.
- Before merge or handoff, contributors MUST run `gofmt` on changed Go files,
  use `goimports` when imports change, execute relevant `go test` commands, and
  record any skipped validation with a reason.
- Code review MUST reject changes that bypass plugin registration, leave
  contract/config drift unresolved, or omit shutdown and measurement-period
  reasoning for concurrent code.

## Governance

- This constitution governs feature planning, implementation, and review across
  the repository. `AGENTS.md` and `doc/go-codex-style.md` remain the Go-specific
  coding companions and MUST be applied in addition to this document.
- Amendments MUST include a rationale tied to repository reality, updates to all
  affected `.specify` templates or operational docs, and a semantic version bump
  for the constitution itself.
- Versioning policy:
  - MAJOR for incompatible principle removals or redefinitions.
  - MINOR for new principles, new mandatory sections, or materially expanded
    governance.
  - PATCH for clarifications, wording improvements, or non-semantic refinements.
- Compliance review expectations:
  - Every plan MUST pass the Constitution Check before design work is approved.
  - Every implementation or review MUST confirm contract/config/deployment sync,
    concurrency and shutdown implications, and verification evidence for the
    affected path.

**Version**: 1.0.0 | **Ratified**: 2026-03-22 | **Last Amended**: 2026-03-22
