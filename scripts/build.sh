#!/usr/bin/env bash
set -euo pipefail

echo "Creating bin directory..."
mkdir -p bin

echo "Building services..."
go build -o bin/ns-api-v1 ./cmd/ns-api/v1
go build -o bin/ns-api-v2 ./cmd/ns-api/v2
go build -o bin/ns-ai ./cmd/ns-ai
go build -o bin/ns-engine ./cmd/ns-engine
go build -o bin/ns-probe ./cmd/ns-probe
go build -o bin/pcap-analyzer ./cmd/pcap-analyzer

echo "Build complete. Binaries are in the ./bin directory."
