# Contract: Cutover Sync Points

## Code Surfaces

| Surface | Paths | Migration Action | Verification Signal |
|--------|-------|------------------|---------------------|
| Contract source | `api/proto/v1/`, `api/thrift/v1/` | Add Thrift IDL, retire Proto as supported source of truth | Thrift IDL exists and supported docs reference only the new root |
| Generated bindings | `api/gen/v1/`, `api/gen/thrift/v1/` | Generate Thrift Go bindings and remove maintained imports of Proto/gRPC outputs | Repository code imports the Thrift bindings or internal DTOs only |
| Packet transport | `internal/probe/`, `pkg/pcap/`, `internal/engine/streamaggregator/`, `internal/engine/manager/` | Replace packet codec and NATS decode/encode path | Offline and NATS packet smokes succeed with Thrift payloads |
| Query RPC | `internal/api/`, `internal/query/`, `scripts/query/v2/` | Replace gRPC server/client flow and remove `protojson` dependency from supported handlers | Query smoke commands succeed through the new RPC path |
| AI RPC | `internal/ai/`, `internal/alerter/`, `scripts/ask-ai/` | Replace gRPC server/client flow and add prompt session handling | Unary and chunked prompt workflows succeed through Thrift RPC |

## Config And Deployment Surfaces

| Surface | Paths | Migration Action | Verification Signal |
|--------|-------|------------------|---------------------|
| Runtime config | `configs/config.yaml`, `configs/config.yaml.example`, `internal/config/config.go` | Rename `grpc_*` runtime keys and update parsing structs | Config load and service startup work with the renamed keys |
| Kubernetes manifests | `deployments/kubernetes/` | Rename listener/service labels and update env wiring | Deployment manifests expose the new RPC names only |
| Helm chart | `deployments/helm/go2netspectra/` | Update values, templates, and service port naming | Helm render shows no active `grpc` runtime surface for supported services |
| Docs | `README.md`, `doc/build.md`, `doc/technology.md`, `doc/refactor/module-boundaries.md` | Replace Proto/gRPC setup and architecture guidance with Thrift guidance | Supported docs describe Thrift generation and RPC usage consistently |

## Runtime Behavior Guarantees

- Packet messages on NATS use exactly one supported contract family after
  cutover: Thrift.
- Query and AI services reject legacy gRPC clients quickly and explicitly.
- Supported HTTP routes, if preserved, do not rely on Protobuf-specific DTO or
  JSON helpers.
- Prompt-analysis sessions have explicit creation, read, cancel, expiry, and
  shutdown semantics.

## Retired Support Surface

The following remain in repository history only and must not be treated as
active supported workflows after cutover:

- Protobuf generation instructions for maintained runtime paths
- gRPC client/server startup instructions for query and AI services
- Active imports of `google.golang.org/grpc`
- Active imports of `google.golang.org/protobuf` outside retired assets or explicit legacy-rejection tests
- Active reliance on `protojson` for supported query/API behavior
