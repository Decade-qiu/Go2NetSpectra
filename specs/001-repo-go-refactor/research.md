# Research: Repository-Wide Go Refactor Program

## Decision 1: Use staged vertical slices instead of a single repository-wide rewrite

**Decision**: Execute the refactor in bounded phases aligned to the existing
runtime pipeline, with each phase independently reviewable and reversible.

**Rationale**: The repository mixes live traffic processing, offline analysis,
query services, alerting, AI analysis, and deployment assets. A single
repository-wide rewrite would hide regressions, inflate review size, and make it
hard to isolate failures. Vertical slices preserve context and allow phase exit
criteria.

**Alternatives considered**:

- **Big-bang rewrite**: Rejected because it combines structure, convention, and
  performance changes into one high-risk delivery.
- **Directory-by-directory cleanup only**: Rejected because it can preserve
  broken cross-layer responsibilities.

## Decision 2: Preserve current public behavior and runtime topology by default

**Decision**: Treat current protobuf semantics, runtime configuration meaning,
supported service startup flows, and externally used query/alert/AI behaviors as
stable unless a specific incompatibility is approved.

**Rationale**: The refactor request is about structure, quality, and
performance. Compatibility drift would turn the project into a feature rewrite
instead of a safe modernization effort.

**Alternatives considered**:

- **Allow incidental behavior changes during cleanup**: Rejected because it
  makes regressions indistinguishable from intended design updates.
- **Freeze only APIs, ignore scripts/deployments**: Rejected because this
  repository is operated through both code and runtime assets.

## Decision 3: Keep generated artifacts derived, not hand-refactored

**Decision**: Refactor the source-of-truth files and regenerate outputs when
necessary. Generated protobuf code remains derived output and is not treated as
an ownership target.

**Rationale**: Hand-editing derived files would create long-term maintenance
drift and hide where true contract ownership lives.

**Alternatives considered**:

- **Treat generated files as primary refactor targets**: Rejected because it
  duplicates effort and weakens contract discipline.
- **Ignore generated outputs completely**: Rejected because compatibility still
  requires regenerated outputs to stay in sync.

## Decision 4: Prioritize concurrency and lifecycle safety as design constraints

**Decision**: Any refactor affecting channels, goroutines, snapshot loops,
reset loops, or shutdown behavior must explicitly preserve ownership, lifecycle,
and final-flush semantics before any performance tuning is accepted.

**Rationale**: The current codebase relies on concurrent packet handling and
periodic snapshotting. Structural cleanup without lifecycle protection would
create subtle correctness regressions.

**Alternatives considered**:

- **Optimize first, document concurrency later**: Rejected because it can hide
  leaks and race conditions behind throughput gains.
- **Restrict refactor to naming and file moves**: Rejected because the request
  explicitly includes performance and structural cleanup.

## Decision 5: Use a validation ladder from fast checks to realistic runtime checks

**Decision**: Validate each phase with four levels of evidence:

1. Focused package and repository regression tests.
2. Representative fixture and script checks.
3. Service-level startup and interaction smoke checks.
4. Benchmark or baseline comparison for prioritized hot paths.

**Rationale**: Fast checks catch most logic regressions early, while runtime
and performance checks confirm that refactors did not silently break operated
paths.

**Alternatives considered**:

- **Repository tests only**: Rejected because tests alone do not prove runtime
  workflows or performance characteristics.
- **Manual smoke checks only**: Rejected because they are too slow and too easy
  to perform inconsistently.

## Decision 6: Focus performance work on cost centers before algorithm changes

**Decision**: Treat allocation pressure, duplicated work, lock contention,
unnecessary conversions, and lifecycle overhead as the first performance targets.
Only revisit algorithm choices if baseline evidence shows they remain the
dominant bottleneck after structural cleanup.

**Rationale**: The codebase already contains domain-specific exact/sketch
implementations and benchmark infrastructure. Most safe wins are likely to come
from lifecycle and organization fixes before algorithm replacement.

**Alternatives considered**:

- **Replace major algorithms immediately**: Rejected because it expands scope
  from refactor into feature redesign.
- **Ignore performance until all cleanup is complete**: Rejected because some
  structural choices directly affect hot-path cost and must be designed upfront.

## Phase 1 Working Inventory

### Touched packages

- `internal/probe`, `pkg/pcap`, and `internal/protocol` for transport packet ownership.
- `internal/engine/manager`, `internal/engine/streamaggregator`, and `internal/engine/app` for runtime orchestration.
- `internal/api`, `internal/query`, `cmd/ns-api/v1`, and `cmd/ns-api/v2` for query/API assembly.
- `internal/ai` and `cmd/ns-ai` for AI service assembly.
- `internal/factory` plus `internal/engine/impl/*` for plugin registration coverage.

### Public and operational surfaces

- Protobuf transport packets under `api/proto/v1/` and generated outputs under `api/gen/v1/`.
- Runtime configuration keys in `configs/config.yaml`.
- Docker Compose and Helm values under `deployments/`.
- Build, lint, smoke, and phase-exit instructions in `doc/`, `README.md`, and `specs/001-repo-go-refactor/`.

### MVP scope

- Phase 1 setup records the boundary map, evidence template, and validation entrypoints.
- Foundational work establishes a shared packet transport seam, package ownership docs, and plugin-registration regression coverage.
- User Story 1 extracts entrypoint assembly into long-term internal packages without changing default public behavior.
