# Go2NetSpectra

A distributed network traffic monitoring and analysis framework.

## Project Structure

The project follows the Standard Go Project Layout.

- `cmd/`: Main applications for the project.
- `internal/`: Private application and library code.
- `pkg/`: Library code that's ok to use by external applications.
- `api/`: API definitions, e.g., Protobuf, OpenAPI.
- `configs/`: Configuration file templates.
- `deployments/`: Deployment scripts and configurations.
- `scripts/`: Scripts to build, install, analyze, etc.
- `test/`: Test data and scripts.

## Getting Started

### Prerequisites

- Go (version 1.21 or later)
- Docker and Docker Compose (for running dependencies like ClickHouse, Kafka)

### Configuration

1.  Copy the example configuration file:
    ```sh
    cp configs/config.yaml.example configs/config.yaml
    ```
2.  Edit `configs/config.yaml` to match your environment.

### Build

To build the binaries for all services, run the build script:

```sh
./scripts/build.sh
```

This will create the binaries in the `bin/` directory.

### Running the services

You can run each service directly using `go run`:

```sh
go run ./cmd/ns-api/main.go
go run ./cmd/ns-engine/main.go
go run ./cmd/ns-probe/main.go
```

Or you can run the compiled binaries:

```sh
./bin/ns-api
./bin/ns-engine
./bin/ns-probe
```
