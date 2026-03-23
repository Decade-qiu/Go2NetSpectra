# [PROJECT NAME] Development Guidelines

Auto-generated from all feature plans. Last updated: [DATE]

## Active Technologies

- Go 1.25.x
- gopacket for packet capture and parsing
- Protobuf + gRPC for inter-service contracts
- NATS for transport
- ClickHouse for queryable storage
- YAML configuration with environment-variable expansion
- Docker Compose, Kubernetes, and Helm for deployment

## Project Structure

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

## Commands

```bash
go test ./...
go test ./internal/protocol ./internal/engine/impl/sketch ./internal/engine/impl/benchmark
go run ./cmd/ns-engine/main.go
go run ./cmd/ns-api/v2/main.go
go run ./cmd/ns-ai/main.go
go run ./cmd/ns-probe/main.go --mode=pub --iface=<iface>
go run ./cmd/pcap-analyzer/main.go <pcap_file>
go run ./scripts/query/v2/main.go --mode=aggregate --task=<task>
protoc --proto_path=api/proto --go_out=. --go-grpc_out=. api/proto/v1/*.proto
```

## Code Style

- Preserve the capture -> protobuf/NATS -> manager -> task group ->
  writer/query/alert/AI layering.
- New aggregators and writers are config-driven and registered with
  `factory.RegisterAggregator` plus the required blank import.
- Keep reusable logic out of `cmd/`; keep private runtime logic in `internal/`.
- Treat `api/proto/v1/` and `configs/config.yaml` as authoritative contracts and
  regenerate/sync derived artifacts after changes.
- Use targeted `go test`, pcap-fixture validation, client-script smoke tests,
  and benchmarks for hot-path changes.

## Recent Changes

[LAST 3 FEATURES AND WHAT THEY ADDED]

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
