#!/usr/bin/env bash

set -euo pipefail

command=${1:-start}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

export INVENTORY_API_PORT="8080"
export INVENTORY_API_MONGODB_USERNAME="root"
export INVENTORY_API_MONGODB_PASSWORD="neUhaDnes"

# Helper function (equivalent to PowerShell function mongo)
mongo() {
  docker compose --file "$PROJECT_ROOT/deployments/docker-compose/compose.yaml" "$@"
}

case "$command" in
  openapi)
    docker run --rm -ti \
      -v "$PROJECT_ROOT:/local" \
      openapitools/openapi-generator-cli \
      generate -c /local/scripts/generator-cfg.yaml
    ;;

  start)
    trap 'mongo down' EXIT

    mongo up --detach
    go run "$PROJECT_ROOT/cmd/inventory-api-service"
    ;;

  test)
    go test -v ./...
    ;;

  docker)
    docker build -t kleeat/inventory-webapi:local-build -f "$PROJECT_ROOT/build/docker/Dockerfile" .
    ;;

  mongo)
    mongo up
    ;;

  *)
    echo "Unknown command: $command"
    exit 1
    ;;
esac