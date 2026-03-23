# Contract: Public And Operational Surfaces

## Purpose

Define which repository-visible behaviors must remain stable by default while
the refactor is in progress.

## Surface 1: Traffic Ingestion Semantics

- Live traffic capture and offline pcap analysis continue to represent packet
  metadata with the same operational meaning.
- Refactors may reorganize ownership and code paths, but they must not silently
  change what a valid packet record means to downstream processing.

## Surface 2: Processing And Snapshot Semantics

- Exact and sketch processing remain part of the supported processing model.
- Snapshot, reset, and shutdown behavior must stay explainable and preserve
  existing correctness guarantees unless a change is explicitly approved.

## Surface 3: Query, Alerting, And AI Availability

- Existing query, alerting, and AI-assisted analysis workflows remain available
  unless the phase explicitly documents a compatible replacement path.
- Refactors must not strand one supported workflow while only modernizing
  another.

## Surface 4: Configuration And Deployment Expectations

- Runtime configuration meaning remains stable by default.
- Supported startup workflows and deployment assets must stay synchronized with
  refactored code ownership.

## Surface 5: Developer Navigation Contract

- After each accepted phase, maintainers must be able to answer:
  - Which boundary owns this behavior?
  - Which file or package should be changed?
  - What public or operational surface could be affected?

If those questions cannot be answered quickly, the phase is not complete.
