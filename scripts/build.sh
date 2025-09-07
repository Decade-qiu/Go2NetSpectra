#!/bin/bash
set -e

echo "Creating bin directory..."
mkdir -p bin

echo "Building services..."
go build -o bin/ns-api ./cmd/ns-api
go build -o bin/ns-engine ./cmd/ns-engine
go build -o bin/ns-probe ./cmd/ns-probe

echo "Build complete. Binaries are in the ./bin directory."
