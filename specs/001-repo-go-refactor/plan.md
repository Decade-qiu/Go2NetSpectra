# Implementation Plan: Repository-Wide Go Refactor Program

**Branch**: `001-repo-go-refactor` | **Date**: 2026-03-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-repo-go-refactor/spec.md`

**Note**: This plan turns the repository-wide Go refactor request into staged,
reviewable work packages that preserve current public behavior by default.

## Summary

Refactor the maintained Go codebase in phases to clarify module ownership across
the existing ingestion -> processing -> query/alert/AI pipeline, normalize
repository-wide Go conventions, and improve hot-path performance without
breaking public contracts or supported runtime workflows. The plan uses staged
vertical slices: boundary cleanup first, convention consolidation second, and
performance hardening third, with compatibility checkpoints after each phase.

## Technical Context

**Language/Version**: Go 1.25.0
**Primary Dependencies**: gopacket, Protobuf/gRPC, NATS, ClickHouse, YAML
configuration, go-openai, gorilla/mux
**Storage**: ClickHouse, optional gob/text/pcap outputs, YAML configuration,
generated protobuf artifacts, and local fixture data under `test/`
**Testing**: `go test ./...`, focused parser/aggregator/query package tests,
pcap-fixture validation, client-script smoke tests, and benchmark coverage in
`internal/engine/impl/benchmark`
**Target Platform**: Linux/macOS development plus Docker Compose and
Kubernetes/Helm deployment environments
**Project Type**: Distributed Go services and CLI utilities for network traffic
monitoring and analysis
**Performance Goals**: Preserve end-to-end correctness while avoiding more than
5% regression on prioritized hot paths; improve allocation, contention, and
lifecycle behavior where feasible
**Constraints**: Preserve public protobuf/config semantics by default; keep
`cmd/` as composition only; retain plugin registration flow; maintain read-only
snapshot semantics and explicit goroutine shutdown; update generated assets and
ops docs when runtime surfaces move
**Scale/Scope**: Repository-wide refactor across maintained Go packages,
service entrypoints, tests, scripts, config/deployment alignment, and phased
compatibility validation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Pipeline layering is preserved. The plan keeps the existing
      capture/parse -> protobuf/NATS -> manager -> task group ->
      writer/query/alert/AI flow and restructures responsibilities within those
      layers instead of inventing a parallel pipeline.
- [x] Plugin registration remains authoritative. Refactors around aggregators
      and writers keep `model.Task`, `model.Writer`,
      `factory.RegisterAggregator`, config-driven creation, and manager blank
      imports as the only supported extension path.
- [x] Contract synchronization is explicit. Any touched public surface must
      update protobuf definitions, generated artifacts, runtime config,
      client/server call sites, and deployment assets together.
- [x] Concurrency design is first-class. Work packages must document goroutine
      ownership, channel lifecycle, snapshot/reset invariants, and graceful
      shutdown behavior before implementation.
- [x] Verification is concrete. The plan includes package tests, pcap-based
      validation, service/client smoke workflows, and benchmark/baseline checks
      for performance-sensitive paths.

## Project Structure

### Documentation (this feature)

```text
specs/001-repo-go-refactor/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── public-surfaces.md
│   └── refactor-exit-criteria.md
└── tasks.md
```

### Source Code (repository root)

```text
api/
├── proto/v1/
└── gen/v1/
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
test/
```

**Structure Decision**: Execute the refactor as vertical slices aligned to the
existing pipeline. Each slice may touch `cmd/`, `internal/`, tests, scripts,
and ops assets, but reusable runtime logic stays out of `cmd/`, generated code
remains derived, and public surfaces are stabilized before deeper internal
movement.

## Phase 0: Research Summary

Research focuses on the safest strategy for a repository-wide refactor:

1. Preserve the current runtime topology and public behavior by default.
2. Refactor by bounded vertical slices instead of a single mega-change.
3. Treat concurrency semantics, generated artifacts, and deployment parity as
   mandatory design concerns, not cleanup work.
4. Use a validation ladder that escalates from package tests to fixture,
   service, and benchmark checks.

See [research.md](./research.md) for the full decision log.

## Phase 1: Design

### Work Packages

1. **Boundary Governance**
   - Clarify ownership of service entrypoints versus shared runtime logic.
   - Reduce duplicate or misplaced logic across probe, engine, query, alerting,
     AI, and script paths.
   - Define which files stay as generated artifacts or runtime wiring only.

2. **Convention Consolidation**
   - Normalize naming, comments, errors, context usage, imports, testing
     patterns, and directory usage to match repository rules.
   - Identify high-risk legacy patterns that should be removed, wrapped, or
     explicitly deferred.

3. **Hot-Path Performance Hardening**
   - Prioritize parser, probe publication/subscription, manager orchestration,
     exact/sketch execution, query access, and shutdown/snapshot loops.
   - Focus on allocation pressure, lock contention, unnecessary duplication, and
     lifecycle leaks before considering deeper algorithm changes.

### Data Model

The design tracks refactor work through bounded phases, module boundary maps,
compatibility decisions, verification suites, and performance baselines. See
[data-model.md](./data-model.md).

### Contracts

The feature exposes repository-level compatibility contracts rather than new
network APIs. The refactor must preserve:

- Ingestion semantics for live and offline packet processing.
- Query, alerting, and AI workflow availability.
- Runtime configuration/deployment expectations.
- Phase exit criteria for what counts as a safe structural change.

See [public-surfaces.md](./contracts/public-surfaces.md) and
[refactor-exit-criteria.md](./contracts/refactor-exit-criteria.md).

### Quickstart Validation

The implementation will be validated through a repeatable workflow covering
baseline capture, refactor checkpoints, service smoke tests, and benchmark
comparison. See [quickstart.md](./quickstart.md).

## Post-Design Constitution Check

- [x] The design keeps all refactors inside the existing layered pipeline and
      defines no new top-level runtime boundary.
- [x] Aggregator and writer refactors remain bound to the existing plugin model
      and configuration-driven creation path.
- [x] Contract/config/deployment synchronization is modeled explicitly in both
      contracts and phase exit criteria.
- [x] Concurrency invariants are represented in the data model and quickstart
      validation path.
- [x] Verification covers package, fixture, service, and benchmark levels.

## Implementation Phases

### Phase 1: Boundary Governance

- Inventory maintained Go code and classify each package as runtime wiring,
  shared logic, generated artifact, test support, or deferred legacy area.
- Remove cross-layer leakage and duplicated orchestration where it obscures the
  main pipeline.
- Land documentation that maps runtime behaviors to clear ownership boundaries.

### Phase 2: Convention Consolidation

- Normalize repository-wide Go style on touched files and packages.
- Add or improve comments and tests where boundary or lifecycle rules would
  otherwise stay implicit.
- Record deferred debt that cannot be safely eliminated in the same phase.

### Phase 3: Hot-Path Performance Hardening

- Establish baseline evidence for prioritized paths.
- Apply targeted optimizations that reduce cost without weakening correctness or
  shutdown guarantees.
- Re-run validation ladder and accept only measured improvements or bounded
  non-regression.

## Complexity Tracking

No constitution violations are currently required. The plan deliberately avoids
introducing a new pipeline, a new extension model, or compatibility-breaking
surface changes as the default strategy.
