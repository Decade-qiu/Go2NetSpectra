# Data Model: Repository-Wide Go Refactor Program

## Entity: RefactorPhase

**Purpose**: Represents a bounded delivery slice of the repository refactor.

**Fields**:

- `name`: Human-readable phase name.
- `objective`: The primary outcome the phase must deliver.
- `owned_areas`: The repository areas covered by the phase.
- `compatibility_scope`: Which runtime surfaces must remain stable.
- `exit_criteria`: The validation conditions required to close the phase.
- `deferred_items`: Explicitly accepted work left for later phases.
- `status`: Proposed, in-progress, validated, or complete.

**Validation Rules**:

- Must map to a bounded set of repository areas.
- Must define at least one compatibility expectation.
- Must define at least one measurable exit criterion.

**Relationships**:

- Owns multiple `ModuleBoundary` records.
- Produces multiple `VerificationSuite` results.
- Can emit multiple `CompatibilityDecision` records.

**State Transitions**:

- `Proposed -> In Progress -> Validated -> Complete`
- `In Progress -> Proposed` if scope must be re-cut before safe delivery

## Entity: ModuleBoundary

**Purpose**: Describes a logical ownership boundary for maintained code.

**Fields**:

- `name`: Boundary name.
- `layer`: Ingestion, transport, orchestration, execution, query, alerting,
  AI, config/deployment, or support tooling.
- `responsibilities`: What the boundary owns.
- `allowed_dependencies`: Which neighboring boundaries it may depend on.
- `forbidden_dependencies`: Which cross-layer shortcuts are not allowed.
- `current_locations`: Existing repository areas currently associated with it.
- `target_ownership_rule`: Desired long-term ownership rule after refactor.

**Validation Rules**:

- Must define both responsibilities and forbidden dependencies.
- Must not overlap ambiguously with another boundary in the same phase.

**Relationships**:

- Belongs to one `RefactorPhase`.
- Can be referenced by multiple `CompatibilityDecision` and
  `VerificationSuite` records.

## Entity: CompatibilityDecision

**Purpose**: Records whether a structural change preserves or adjusts an
existing public or operational surface.

**Fields**:

- `surface`: The affected public or operational surface.
- `current_behavior`: Existing expected behavior.
- `target_behavior`: Intended behavior after the phase.
- `compatibility_mode`: Preserved, migrated, or intentionally changed.
- `migration_note`: Required operator or developer action if behavior changes.
- `approval_status`: Proposed, accepted, or rejected.

**Validation Rules**:

- Must identify the surface being protected.
- Must include migration guidance whenever compatibility is not preserved.

**Relationships**:

- Can be attached to one or more `RefactorPhase` records.
- Must be referenced by a `VerificationSuite` if the change affects runtime use.

## Entity: VerificationSuite

**Purpose**: Defines the evidence required to prove a refactor phase is safe.

**Fields**:

- `name`: Validation suite name.
- `level`: Repository, fixture, service-smoke, or benchmark.
- `target_path`: The path or workflow being validated.
- `purpose`: What regression or promise it protects.
- `success_signal`: Observable result required to pass.
- `baseline_source`: Existing measurement or expected behavior reference.

**Validation Rules**:

- Must identify a clear target path and success signal.
- Benchmark suites must name a baseline source.

**Relationships**:

- Belongs to one `RefactorPhase`.
- Can reference multiple `ModuleBoundary` or `CompatibilityDecision` records.

## Entity: PerformanceBaseline

**Purpose**: Tracks the before/after measurement source for prioritized hot
paths.

**Fields**:

- `path_name`: Hot path under measurement.
- `metric_family`: Throughput, latency, allocations, contention, or shutdown cost.
- `baseline_value`: Pre-refactor observed value.
- `target_rule`: Improvement target or non-regression rule.
- `evidence_reference`: Where the measurement result is recorded.

**Validation Rules**:

- Must include a measurable metric family.
- Must define either an improvement target or acceptable non-regression bound.

**Relationships**:

- Can be referenced by one or more `VerificationSuite` records.
- Typically belongs to a `RefactorPhase` focused on hot-path performance.
