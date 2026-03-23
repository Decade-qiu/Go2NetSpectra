# Go2NetSpectra Module Boundaries

This document records the repository ownership rules used by the repository-wide Go refactor MVP.

## Boundary Map

| Layer | Owned Paths | Responsibility | Must Not Own |
|---|---|---|---|
| Capture and parsing | `cmd/ns-probe`, `pkg/pcap`, `internal/protocol`, `internal/probe` | Capture packets, parse protocol metadata, encode/decode transport packets, and optionally persist probe-side artifacts | Query routing, aggregation state, alert evaluation |
| Transport contracts | `api/thrift/v1`, `api/gen/thrift/v1` | Define Thrift transport schemas and generated client/server code | Runtime business logic or manual hotfixes in generated files |
| Runtime orchestration | `internal/engine/manager`, `internal/engine/streamaggregator`, `internal/engine/app` | Start and stop runtime flows, own packet fan-out, snapshot/reset cadence, and offline engine bootstrapping | Packet capture details or storage query policies |
| Execution backends | `internal/engine/impl/exact`, `internal/engine/impl/sketch`, `internal/factory`, `internal/model` | Register aggregators, construct task groups from config, and process packets into snapshots/queryable state | CLI wiring or deployment manifests |
| Query and API | `internal/api`, `internal/query`, `cmd/ns-api/v1`, `cmd/ns-api/v2` | Assemble HTTP/Thrift query servers and execute query workloads against persisted data | Probe capture logic or aggregator internals |
| Alerting and AI | `internal/alerter`, `internal/ai`, `cmd/ns-ai` | Evaluate rules, notify operators, and provide AI-backed enrichment | Probe ingestion, packet parsing, or deployment-specific defaults |
| Operations surface | `configs/`, `deployments/`, `doc/`, `README.md`, `specs/001-repo-go-refactor/` | Keep config, runtime docs, validation steps, and deployment assets aligned with the code | Hidden behavior changes without recorded compatibility notes |

## Target Package Additions

| Package | Long-term role |
|---|---|
| `internal/api` | Shared server assembly for retired legacy HTTP handling, Thrift query RPC, and Grafana-compatible HTTP routes |
| `internal/engine/app` | Shared runtime assembly for stream engine startup and offline analyzer bootstrapping |
| `doc/refactor` | Human-readable ownership maps, deferred debt notes, and refactor phase outcomes |

## Entrypoint Guidance

| Entrypoint | Wiring only | Shared runtime home |
|---|---|---|
| `cmd/ns-api/v1/main.go` | Load config, install signal handling, delegate to shared server runner | `internal/api/http_server.go` |
| `cmd/ns-api/v2/main.go` | Load config, install signal handling, delegate to shared server runner | `internal/api/grpc_server.go` and `internal/api/http_server.go` |
| `cmd/ns-ai/main.go` | Parse flags, load config, install signal handling, delegate to server runner | `internal/ai/server.go` |
| `cmd/ns-engine/main.go` | Load config, install signal handling, delegate to runtime runner | `internal/engine/app/runner.go` |
| `cmd/pcap-analyzer/main.go` | Parse input path, load config, delegate to offline runner | `internal/engine/app/runner.go` |

## Representative Path Traces

### Live probe to engine to query

1. `cmd/ns-probe/main.go` owns process wiring only.
2. `internal/protocol` parses raw packets into `model.PacketInfo`.
3. `internal/probe` converts packets into Thrift transport payloads and publishes to NATS.
4. `internal/engine/streamaggregator` consumes Thrift packets and forwards them to `internal/engine/manager`.
5. `internal/engine/impl/*` tasks update exact or sketch state and writers persist snapshots.
6. `internal/query` serves persisted results through `internal/api`.

### Offline analyzer to engine to query

1. `cmd/pcap-analyzer/main.go` delegates startup to `internal/engine/app`.
2. `pkg/pcap` reads offline traffic and emits domain packets for the shared manager path.
3. `internal/engine/manager` routes decoded packets to registered tasks.
4. `internal/query` and `internal/api` expose the resulting data.

### Alerting and AI

1. `internal/engine/manager` owns task execution cadence only.
2. `internal/alerter` evaluates rules against task outputs.
3. `internal/ai` exposes the AI Thrift RPC service used for enrichment.
4. `cmd/ns-ai/main.go` remains process wiring only.

## Smoke Workflows

### Offline path

```bash
go test ./internal/protocol ./internal/probe ./internal/engine/manager
go run ./cmd/pcap-analyzer/main.go test/data/test.pcap
```

### Live probe path

```bash
go run ./cmd/ns-engine/main.go
go run ./cmd/ns-probe/main.go --mode=pub --iface=<interface_name>
go run ./cmd/ns-probe/main.go --mode=sub
```

### Query and API path

```bash
go run ./cmd/ns-api/v2/main.go
go run ./scripts/query/v2/main.go --mode=aggregate --task=per_five_tuple
```

### Alerting and AI path

```bash
go run ./cmd/ns-ai/main.go
go run ./scripts/ask-ai/main.go "Summarize the latest network anomalies"
```

Validate alerting by confirming the engine and AI services are both reachable and that any enabled rule produces a recorded notification or AI enrichment log before phase sign-off.

## Deferred Smoke Gaps

- Offline analyzer smoke is currently the only fully executed end-to-end runtime smoke in the local refactor loop.
- Live probe smoke still depends on a local NATS broker plus a reproducible publish helper that does not require manual interface capture.
- Query/API and AI smoke still depend on local ClickHouse readiness and AI credentials, so they remain explicit sign-off items rather than silent assumptions.
