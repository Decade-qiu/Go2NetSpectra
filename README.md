# Go2NetSpectra

[![Go](https://img.shields.io/badge/go-1.25-blue.svg)](https://go.dev/) [![gopacket](https://img.shields.io/badge/gopacket-1.1.19-blue.svg)](https://github.com/google/gopacket) [![NATS](https://img.shields.io/badge/NATS-2.11-green.svg)](https://nats.io/) [![Protobuf](https://img.shields.io/badge/Protobuf-v3-blue.svg)](https://protobuf.dev/) [![Docker](https://img.shields.io/badge/docker-20.10%2B-blue)](https://www.docker.com/)

**Go2NetSpectra** is a high-performance, distributed network traffic monitoring and analysis framework written in Go. It provides a powerful platform for network engineers, security analysts, and SREs to gain deep, multi-dimensional insights into network traffic in real-time. By leveraging a high-speed data pipeline and a flexible, pluggable aggregation engine, Go2NetSpectra is built to scale from simple network monitoring to complex security threat detection.

### Core Features

- **Hybrid Analysis Engine**: Simultaneously runs multiple aggregator types. This allows the system to perform **100% accurate accounting** (`exact` mode) and **high-performance probabilistic analysis** (`sketch` mode) *at the same time*, enabling powerful, data-driven workflows (e.g., use `sketch` to find anomalies, then use `exact` to get precise details).
- **Real-time Alerting**: A built-in alerting pipeline allows tasks to generate event messages (e.g., heavy hitter detected). These are processed by a central `Alerter` which can trigger notifications via webhooks, providing immediate insights into network events.
- **Pluggable Aggregation Algorithms**: The `sketch` aggregator is a micro-framework that dynamically loads different estimation algorithms based on configuration. Currently supports **Count-Min Sketch** (for heavy hitters) and **SuperSpread** (for cardinality/super-spreaders).
- **High-Performance by Design**: Built from the ground up for performance, utilizing Go's concurrency model (worker pools), lock-free optimizations (atomic operations in sketches), and efficient data serialization (Protobuf).
- **Decoupled & Scalable**: All major components (`probe`, `engine`, `api`) are decoupled via a message bus and are designed to be horizontally scalable, making the system suitable for high-volume, distributed environments.

---

## Architecture Overview

Go2NetSpectra operates as a multi-stage, distributed pipeline designed for performance, scalability, and real-time analysis.

```mermaid
graph TD
    subgraph "Data Plane"
        direction LR
        Iface[Network Interface] -- live traffic --> Probe[ns-probe]
        Pcap[PCAP File] -- offline traffic --> Analyzer[pcap-analyzer]
        Probe -- Protobuf over NATS --> NATS[(NATS Message Bus)]
    end

    subgraph "Processing Plane (ns-engine)"
        direction TB
        NATS -- Protobuf --> Manager(Manager: Worker Pool)
        
        subgraph "Aggregation Tasks"
            Manager -- fan-out --> Tasks(Sketch & Exact Tasks)
        end

        Tasks -- snapshot --> Storage(Storage: ClickHouse)
        
        subgraph "Real-time Alerting"
            Tasks -- generates event --> Manager
            Manager -- forwards --> Alerter(Alerter)
            Alerter --> Notifier(Notifier: Webhook)
        end
    end

    subgraph "Query Plane (ns-api)"
        API[ns-api]
        User[User/Client] -- gRPC --> API
        API -- queries --> Storage
    end

    style NATS fill:#FFB6C1,stroke:#333,stroke-width:2px
    style Manager fill:#ADD8E6,stroke:#333,stroke-width:2px
    style API fill:#90EE90,stroke:#333,stroke-width:2px
    style Alerter fill:#FFD700,stroke:#333,stroke-width:2px
```

- **Data Sources**: The system processes both live traffic via `ns-probe` and offline `pcap` files using `pcap-analyzer`.
- **Pipeline**: Live data is serialized with Protobuf and streamed through NATS, decoupling the probe from the processing engine.
- **Engine Core**: The heart of the system, where the `Manager` orchestrates a worker pool. It fans out incoming data to various pluggable `Task` aggregators (like `Exact` and `Sketch`) for parallel processing.
- **Persistence & Alerting**: Aggregated data is periodically snapshotted to a ClickHouse database. Simultaneously, tasks can generate real-time events (like detecting a heavy hitter), which are routed through the `Alerter` to trigger external notifications.
- **Query & Visualization**: The `ns-api` server provides a gRPC endpoint for programmatic queries and an HTTP/JSON endpoint for visualization tools like Grafana, which queries the aggregated data from ClickHouse.

For a more detailed explanation of the architecture, configuration files (`config.yaml` vs `config.docker.yaml`), and how to run validation tests, see [`doc/technology.md`](doc/technology.md) and [`doc/build.md`](doc/build.md).

---

## Getting Started

(The rest of the README remains the same)

This guide provides two primary ways to run the project. Choose the one that best fits your needs.

### Prerequisites

- Go 1.25+
- `protoc` Compiler
- Docker and Docker Compose

### First-Time Setup (Protobuf Generation)

This step is only required once, or whenever you modify a `.proto` file in the `api/proto/v1/` directory.
```sh
# Install Go plugins for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

# Generate Go code
protoc --proto_path=api/proto \
       --go_out=. --go-grpc_out=. \
       api/proto/v1/*.proto
```

---

### Option 1: Run with Docker Compose (Recommended)

This is the easiest way to run the entire backend system. You will run all backend services (`nats`, `clickhouse`, `ns-engine`, `ns-api`) in Docker, and then run `ns-probe` on your local machine to capture and send traffic.

**Step 1: Configure for Local Probe**

Ensure your **`configs/config.yaml`** is configured for your local `ns-probe` to connect to the Dockerized NATS service. The `probe` section should point to `localhost`.

```yaml
# configs/config.yaml
probe:
  nats_url: "nats://localhost:4222"
  # ...
```

**Step 2: Start Backend Services**

Navigate to the Docker Compose directory and start all services. This uses `configs/config.docker.yaml` internally for container-to-container communication.

```sh
cd deployments/docker-compose/
docker compose up --build
```
Leave this terminal running.

**Step 3: Capture Traffic on Host**

Open a **new terminal**. Run `ns-probe` locally to capture traffic and send it to the NATS container.

```sh
# Replace <interface_name> with your network interface (e.g., en0, eth0)
sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
```

**Step 4: Query the API**

Open a **third terminal** and use the new **v2 query script** to interact with the `ns-api` gRPC service.

```sh
# Example: Query for aggregated flows
go run ./scripts/query/v2/main.go --mode=aggregate --task=per_src_ip

# Example: Query for heavy hitters detected by a sketch task
go run ./scripts/query/v2/main.go --mode=heavyhitters --task=per_src_ip --type=0 --limit=10
```

---

### Option 2: Run Locally for Development

This mode is useful for debugging individual components (`ns-probe`, `ns-engine`, `ns-api`) directly on your machine, while still using Docker for external dependencies.

**Step 1: Start Dependencies in Docker**

```sh
# Terminal 1: Start NATS
docker run --rm -p 4222:4222 nats:latest

# Terminal 2: Start ClickHouse (note the port mapping 19000:9000)
docker run -d -p 18123:8123 -p 19000:9000 -e CLICKHOUSE_PASSWORD=123 --name some-clickhouse-server --ulimit nofile=262144:262144 clickhouse/clickhouse-server
```

**Step 2: Configure for Localhost**

Ensure your **`configs/config.yaml`** is configured for all services to connect to `localhost`.

```yaml
# configs/config.yaml
probe:
  nats_url: "nats://localhost:4222"
  # ...

aggregator:
  exact:
    writers:
      - type: "clickhouse"
        clickhouse:
          host: "localhost"
          port: 19000
          password: "123"
          # ...

api:
  grpc_listen_addr: ":50051"
  http_listen_addr: ":8080"
```

**Step 3: Run Go Applications Locally**

Open a separate terminal for each command.

```sh
# Terminal 3: Start the Engine
go run ./cmd/ns-engine/main.go

# Terminal 4: Start the API Server (v2)
go run ./cmd/ns-api/v2/main.go

# Terminal 5: Start the Probe
sudo go run ./cmd/ns-probe/main.go --mode=probe --iface=<interface_name>
```