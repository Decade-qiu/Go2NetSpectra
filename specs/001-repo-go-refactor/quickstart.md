# Quickstart: Validate A Refactor Phase

## 1. Establish a Baseline

Run the repository’s standard regression and benchmark checks before touching a
phase-targeted hot path so later measurements have a comparison point.

Suggested baseline workflow:

```bash
go test ./...
go test -bench=. ./internal/engine/impl/benchmark/
```

Record the baseline before code changes:

| Path | Baseline Evidence | Post-Change Evidence | Result | Notes |
|---|---|---|---|---|
| Parser and packet transport | `BenchmarkProtocolParsePacketInto`: `13.50 ns/op`, `0 B/op`, `0 allocs/op` | `go test ./internal/probe ./internal/protocol` | pass | Includes byte-level packet codec round trip and fixture-driven parser coverage |
| Manager lifecycle and snapshot/reset | `go test ./internal/engine/manager ./internal/probe/persistent ./internal/engine/impl/sketch` | `go run ./cmd/pcap-analyzer/main.go test/data/test.pcap` | pass | Offline flow completed; final snapshot/reset and shutdown path exercised |
| Query/API path | pending | pending | pending | Requires local ClickHouse availability for end-to-end smoke |
| Alerting and AI path | pending | pending | pending | Requires reachable AI backend and alerting side effects to validate fully |
| Exact/sketch benchmarks | `Exact`: `298.4 ns/op`; `CountMin`: `101.4 ns/op`; `SuperSpread`: `226.1 ns/op` | `go test -run '^$' -bench '^BenchmarkMurmurHash3RepresentativeFlowInputs$' -benchmem ./scripts/hash` | pass | Hash baseline captured for 16B/37B/74B flow inputs |

## 2. Validate Boundary-Governance Changes

After boundary cleanup, confirm that the repository still supports both offline
and service-oriented workflows.

Suggested checks:

```bash
go test ./internal/protocol ./internal/query ./internal/engine/manager
go run ./cmd/pcap-analyzer/main.go test/data/test.pcap
go run ./cmd/ns-engine/main.go
go run ./cmd/ns-probe/main.go --mode=sub
```

## 3. Validate Convention-Consolidation Changes

After style and structure normalization, confirm repository-wide checks still
pass and the main runtime paths are unaffected.

Suggested checks:

```bash
go test ./...
go run ./cmd/ns-engine/main.go
go run ./cmd/ns-api/v2/main.go
go run ./cmd/ns-ai/main.go
```

## 4. Validate Query And Interaction Paths

Use the repository’s client scripts or equivalent workflows to confirm query
and AI interaction paths remain usable after refactor.

Suggested checks:

```bash
go run ./scripts/query/v2/main.go --mode=aggregate --task=per_five_tuple
go run ./scripts/ask-ai/main.go "Summarize the latest network anomalies"
```

For alerting validation, confirm that:

- `ns-engine` starts with `alerter.enabled: true`
- `ns-ai` is reachable at `ai.grpc_listen_addr`
- An enabled rule produces a recorded notification or AI-enrichment log

## 5. Validate Hot-Path Performance

After performance-sensitive refactors, re-run the relevant benchmark or fixture
workflow and compare against the recorded baseline.

Suggested checks:

```bash
go test -bench=. ./internal/engine/impl/benchmark/
go test ./internal/engine/impl/sketch ./internal/protocol
```

## 6. Record The Phase Result

Before closing a phase, capture:

- What boundary or convention changed
- Which runtime surfaces were protected
- Which validation evidence passed
- Which items were intentionally deferred
- Where the before/after benchmark evidence was recorded
