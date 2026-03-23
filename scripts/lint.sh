#!/usr/bin/env bash
set -euo pipefail

targets=("$@")
if [ "${#targets[@]}" -eq 0 ]; then
  targets=(
    cmd/ns-ai
    cmd/ns-api/v1
    cmd/ns-api/v2
    cmd/ns-engine
    cmd/ns-probe
    cmd/pcap-analyzer
    internal/alerter
    internal/ai
    internal/api
    internal/config
    internal/engine/app
    internal/engine/impl
    internal/engine/manager
    internal/engine/streamaggregator
    internal/factory
    internal/model
    internal/probe
    internal/protocol
    internal/query
    pkg/pcap
    scripts/ask-ai
    scripts/hash
    scripts/query/v1
    scripts/query/v2
  )
fi

go_files=()
while IFS= read -r file; do
  go_files+=("$file")
done < <(find "${targets[@]}" -type f -name '*.go' | sort)

if [ "${#go_files[@]}" -eq 0 ]; then
  echo "No Go files found to lint."
  exit 0
fi

invalid_case_files=()
for file in "${go_files[@]}"; do
  base_name="$(basename "$file")"
  lower_name="$(printf '%s' "$base_name" | tr '[:upper:]' '[:lower:]')"
  if [ "$base_name" != "$lower_name" ]; then
    invalid_case_files+=("$file")
  fi
done

if [ "${#invalid_case_files[@]}" -ne 0 ]; then
  echo "The following Go files must use lowercase filenames:"
  printf '%s\n' "${invalid_case_files[@]}"
  exit 1
fi

unformatted="$(gofmt -l "${go_files[@]}")"
if [ -n "$unformatted" ]; then
  echo "The following Go files need gofmt:"
  echo "$unformatted"
  exit 1
fi

echo "gofmt check passed."

if command -v goimports >/dev/null 2>&1; then
  echo "goimports is available for import normalization."
else
  echo "goimports not found; skipping import normalization check."
fi

go test ./internal/config ./internal/query ./internal/probe ./internal/engine/manager ./internal/factory ./internal/probe/persistent
go test ./internal/alerter ./internal/ai ./internal/api ./internal/engine/app ./internal/engine/streamaggregator ./internal/model
go test ./internal/engine/impl/... ./internal/protocol
go test ./cmd/... ./scripts/ask-ai ./scripts/hash ./scripts/query/v1 ./scripts/query/v2

echo "Lint checks completed successfully."
