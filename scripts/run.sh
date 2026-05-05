#!/usr/bin/env bash

set -e

command=${1:-start}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

export INVENTORY_API_ENVIRONMENT="Development"
export INVENTORY_API_PORT="8080"

case "$command" in
  start)
    go run "$PROJECT_ROOT/cmd/inventory-api-service"
    ;;
  openapi)
    docker run --rm -ti \
      -v "$PROJECT_ROOT:/local" \
      openapitools/openapi-generator-cli \
      generate -c /local/scripts/generator-cfg.yaml
    ;;
  *)
    echo "Unknown command: $command"
    exit 1
    ;;
esac