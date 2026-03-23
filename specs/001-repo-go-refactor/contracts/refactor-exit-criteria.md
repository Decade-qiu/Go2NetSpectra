# Contract: Refactor Phase Exit Criteria

## Purpose

Define what must be true before any refactor phase is considered complete.

## Exit Criteria

### 1. Scope Closure

- The phase touches only the repository areas declared in planning artifacts, or
  any additions are explicitly recorded and justified.

### 2. Ownership Clarity

- Updated code and docs make the new ownership boundary understandable without
  relying on historical tribal knowledge.

### 3. Compatibility Protection

- Any affected public or operational surface is either preserved or explicitly
  documented as changed with migration guidance.

### 4. Validation Evidence

- Required regression, fixture, service-smoke, and benchmark evidence for the
  phase is recorded and reviewed.

| Suite | Path | Baseline | After | Result | Notes |
|---|---|---|---|---|---|
| Repository regression | `./scripts/lint.sh`, `./scripts/build.sh`, `go test ./...` | current mainline state before US2/US3 edits | same commands re-run after refactor slice | pass | `git diff --check` also passed |
| Offline/live smoke | `go run ./cmd/pcap-analyzer/main.go test/data/test.pcap` | no accepted smoke record before this phase | offline analyzer completed successfully | partial | Offline path passed; live NATS smoke still depends on local broker availability |
| Query/API/AI smoke | pending | pending | pending | pending | Requires ClickHouse plus AI backend credentials to exercise end-to-end |
| Alerting workflow | pending | pending | pending | pending | Requires notifier side effects and an enabled rule firing in a live environment |
| Hot-path benchmark | `BenchmarkProtocolParsePacketInto`, `BenchmarkPacketCodecRoundTrip`, `BenchmarkExactTaskProcessPacket`, `BenchmarkCountMinTaskProcessPacket`, `BenchmarkSuperSpreadTaskProcessPacket`, `BenchmarkMurmurHash3RepresentativeFlowInputs` | baselines recorded below | benchmarks re-run after refactor slice | pass | Current results are the accepted baseline for future phases |

### 4.1 Accepted Benchmark Baselines

Captured on March 23, 2026 on an Apple M3 development machine. Unless a phase
explicitly re-baselines a path, benchmark regressions should stay within 5% of
the accepted baseline and must not add new allocations on zero-allocation paths.

| Benchmark | Command | Accepted Baseline | Guardrail |
|---|---|---|---|
| Parser fixture decode | `go test -run '^$' -bench '^BenchmarkProtocolParsePacketInto$' -benchmem ./internal/engine/impl/benchmark` | `13.50 ns/op, 0 B/op, 0 allocs/op` | `<= 14.18 ns/op, 0 B/op, 0 allocs/op` |
| Packet codec round trip | `go test -run '^$' -bench '^BenchmarkPacketCodecRoundTrip$' -benchmem ./internal/engine/impl/benchmark` | `619.3 ns/op, 576 B/op, 13 allocs/op` | `<= 650.3 ns/op, <= 576 B/op, <= 13 allocs/op` |
| Exact task process packet | `go test -run '^$' -bench '^BenchmarkExactTaskProcessPacket$' -benchmem ./internal/engine/impl/benchmark` | `298.4 ns/op, 374 B/op, 4 allocs/op` | `<= 313.3 ns/op, <= 374 B/op, <= 4 allocs/op` |
| Count-Min task process packet | `go test -run '^$' -bench '^BenchmarkCountMinTaskProcessPacket$' -benchmem ./internal/engine/impl/benchmark` | `101.4 ns/op, 48 B/op, 2 allocs/op` | `<= 106.5 ns/op, <= 48 B/op, <= 2 allocs/op` |
| SuperSpread task process packet | `go test -run '^$' -bench '^BenchmarkSuperSpreadTaskProcessPacket$' -benchmem ./internal/engine/impl/benchmark` | `226.1 ns/op, 48 B/op, 2 allocs/op` | `<= 237.5 ns/op, <= 48 B/op, <= 2 allocs/op` |
| MurmurHash3 representative flow inputs | `go test -run '^$' -bench '^BenchmarkMurmurHash3RepresentativeFlowInputs$' -benchmem ./scripts/hash` | `16B: 4.588 ns/op; 37B: 11.82 ns/op; 74B: 21.86 ns/op` | `16B <= 4.82 ns/op; 37B <= 12.41 ns/op; 74B <= 22.95 ns/op` |

### 5. Deferred Debt Disclosure

- Anything intentionally postponed is listed explicitly so later phases do not
  inherit hidden unfinished work.

Current deferred items:

- Live-probe smoke still needs a reproducible local NATS harness or a checked-in smoke helper to validate publish/subscribe without manual interface capture.
- Query/API smoke still needs a fast local ClickHouse fixture or container recipe with predictable readiness for automated validation.
- Alerting and AI smoke still need environment-safe credentials plus an observable notification sink so AI-enrichment and notifier side effects can be asserted automatically.

### 6. Reviewability

- The phase can be reviewed as a coherent, bounded delivery rather than a
  repository-wide “misc cleanup” bundle.
