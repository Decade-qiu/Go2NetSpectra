# Implementation Plan: [FEATURE]

**Branch**: `[###-feature-name]` | **Date**: [DATE] | **Spec**: [link]
**Input**: Feature specification from `/specs/[###-feature-name]/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

[Extract from feature spec: primary requirement, touched pipeline stages, and
technical approach]

## Technical Context

**Language/Version**: Go 1.25.x
**Primary Dependencies**: gopacket, Protobuf/gRPC, NATS, ClickHouse, YAML
configuration, go-openai
**Storage**: ClickHouse, optional gob/text/pcap file outputs, NATS as transport
**Testing**: `go test ./...`, focused package tests, pcap-fixture validation,
client-script smoke tests, and benchmarks in `internal/engine/impl/benchmark`
when hot paths change
**Target Platform**: Linux/macOS development and containerized Linux for Docker,
Helm, and Kubernetes deployments
**Project Type**: Distributed Go services, CLI tools, and deployment assets
**Performance Goals**: Preserve packet-processing throughput and bounded
snapshot/query overhead for the affected pipeline stage
**Constraints**: Preserve `cmd/`/`internal/`/`pkg/` boundaries, config-driven
plugin registration, read-only snapshots, graceful shutdown, and
environment-driven secrets
**Scale/Scope**: Multi-service traffic monitoring pipeline supporting both
offline pcap analysis and real-time NATS ingestion

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [ ] Pipeline layering is preserved: capture/parse -> protobuf/NATS ->
      manager -> task group -> writer/query/alert/AI.
- [ ] New aggregators or writers use `model.Task` / `model.Writer`,
      `factory.RegisterAggregator`, config entries, and the required blank
      import in `internal/engine/manager`.
- [ ] Contract changes list all required sync points across `api/proto/v1/`,
      `api/gen/v1/`, handlers/clients, `configs/config.yaml`, and
      `deployments/`.
- [ ] Concurrency design documents goroutine ownership, channel lifecycle,
      snapshot/reset behavior, and graceful shutdown semantics.
- [ ] Verification names exact commands, fixtures, scripts, benchmarks, and
      deployment smoke checks required for the touched path.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
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

**Structure Decision**: Document the exact binaries, packages, contracts,
configs, scripts, and deployment assets touched by the feature. Keep reusable
logic out of `cmd/`, keep private runtime logic in `internal/`, and treat
generated protobuf code as derived output.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., New pipeline boundary] | [current need] | [why an existing layer could not absorb it] |
| [e.g., Additional goroutine or ticker] | [specific runtime problem] | [why synchronous or existing lifecycle handling was insufficient] |
