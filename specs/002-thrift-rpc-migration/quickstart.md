# Quickstart: Repository-Wide Thrift Contract Migration

## Prerequisites

- Install the Apache Thrift compiler and ensure `thrift` is on `PATH`.
- Provide the same runtime dependencies already required by the repository:
  NATS, ClickHouse, and any AI provider credentials used by `ns-ai`.
- Ensure the feature branch is checked out and the Thrift IDL files exist under
  `api/thrift/v1/`.

## 1. Regenerate Thrift Go Bindings

From the repository root, regenerate the supported contract bindings:

```bash
thrift --gen go -out api/gen/thrift api/thrift/v1/traffic.thrift
thrift --gen go -out api/gen/thrift api/thrift/v1/query.thrift
thrift --gen go -out api/gen/thrift api/thrift/v1/ai.thrift
rm -rf api/gen/thrift/v1/*-remote
```

Confirm the generated outputs land under `api/gen/thrift/v1/` and that the
legacy Proto/gRPC outputs are no longer required for supported workflows.
Also update `doc/build.md` if the generation command or output layout changes.

## 2. Run Focused Contract And Runtime Tests

```bash
go test ./internal/probe ./internal/engine/manager ./internal/engine/streamaggregator
go test ./internal/api ./internal/query ./internal/ai ./internal/alerter
go test ./pkg/pcap ./cmd/ns-api/... ./cmd/ns-ai/... ./scripts/...
go test ./...
```

If packet codec or AI session behavior changes materially, also run the closest
relevant benchmark or fixture validation before sign-off.

## 3. Smoke The Packet Pipeline

Run the offline packet path with a fixture pcap:

```bash
go run ./cmd/pcap-analyzer/main.go test/data/test.pcap
```

For the real-time path, start the engine and probe against an available NATS
broker:

```bash
go run ./cmd/ns-engine/main.go
go run ./cmd/ns-probe/main.go --mode=pub --iface=<interface_name>
go run ./cmd/ns-probe/main.go --mode=sub
```

Success means the upgraded binaries exchange Thrift packet payloads and no
supported runtime path depends on Protobuf decoding.

## 4. Smoke Query And AI RPC

Start the supported services:

```bash
go run ./cmd/ns-api/v2/main.go
go run ./cmd/ns-ai/main.go
```

Run the repository-provided clients:

```bash
go run ./scripts/query/v2/main.go --mode=aggregate --task=per_five_tuple
go run ./scripts/query/v2/main.go --mode=trace --task=per_five_tuple --key="SrcIP=1.2.3.4,DstPort=443"
go run ./scripts/ask-ai/main.go "Summarize the latest network anomalies"
```

Success means the clients complete over the new Thrift RPC surface, including
incremental prompt output through the replacement session/chunk workflow.
The supported default prompt-session behavior is:

- `ReadPromptChunks` blocks until at least one chunk or terminal state is available.
- `CancelPromptAnalysis` is idempotent for an existing session.
- Idle prompt sessions expire after 10 minutes by default.

If `ask-ai` reaches the Thrift session flow but returns an upstream provider
error such as an expired API key, treat that as credential rotation work rather
than a transport regression.

## 5. Verify Supported Ops Assets

Inspect the maintained runtime surface for legacy Proto/gRPC dependencies:

```bash
rg -n "google\\.golang\\.org/(grpc|protobuf)|protojson|grpc_listen_addr|name: grpc" \
  internal cmd scripts configs deployments README.md doc \
  -g '!**/*_test.go' -S
```

Any remaining matches must be justified as historical or explicitly retired
content rather than part of the supported runtime workflow.

## 6. Implementation Validation Notes

- `go test ./...` and the static repository scan above are the minimum local
  sign-off steps for this migration.
- Live NATS, ClickHouse, and deployment smoke runs still require external
  infrastructure. If that infrastructure is not available in the current
  environment, treat Steps 3 through 5 as required follow-up validation rather
  than silently assuming success.
