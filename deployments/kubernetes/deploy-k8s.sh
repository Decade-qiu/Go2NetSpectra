#!/bin/bash

# This script deploys the Go2NetSpectra system to Kubernetes using raw manifest files.
# It respects the dependency order to ensure a smooth startup.
# This script is intended to be run from the 'deployments/kubernetes/' directory.

# Exit immediately if a command exits with a non-zero status.
set -e

# --- 1. Apply Configurations and Secrets ---
echo "Applying base configurations and secrets..."
kubectl apply -f go2netspectra-config.yaml
kubectl apply -f go2netspectra-secret.yaml

# --- 2. Apply Backend Services (NATS & ClickHouse) ---
echo "
Applying NATS components..."
kubectl apply -f nats/nats-config.yaml
kubectl apply -f nats/nats-service.yaml
kubectl apply -f nats/nats-statefulset.yaml

echo "
Applying ClickHouse components..."
kubectl apply -f clickhouse/clickhouse-config.yaml
kubectl apply -f clickhouse/clickhouse-service.yaml
kubectl apply -f clickhouse/clickhouse-statefulset.yaml

# --- 3. Wait for StatefulSets to be Ready ---
# This is a crucial step to ensure dependencies are met.
echo "
Waiting for NATS cluster to be ready... (This may take a few minutes)"
kubectl rollout status statefulset/nats --timeout=5m

echo "
Waiting for ClickHouse cluster to be ready... (This may take a few minutes)"
kubectl rollout status statefulset/clickhouse --timeout=5m

# --- 4. Apply Application Services ---
echo "
Applying application components (ns-engine, ns-api, ns-ai)..."
kubectl apply -f ns-engine/
kubectl apply -f ns-api/
kubectl apply -f ns-ai/

# --- 5. Wait for Deployments to be Ready ---
echo "
Waiting for application deployments to be ready..."
kubectl rollout status deployment/ns-engine
kubectl rollout status deployment/ns-api
kubectl rollout status deployment/ns-ai

echo "
✅ Go2NetSpectra deployment completed successfully!"