...
$env:INVENTORY_API_PORT="8080"
$env:INVENTORY_API_MONGODB_USERNAME="root"
$env:INVENTORY_API_MONGODB_PASSWORD="neUhaDnes"

function mongo {
    docker compose --file ${ProjectRoot}/deployments/docker-compose/compose.yaml $args
}

switch ($command) {
    "openapi" {
        docker run --rm -ti  -v ${ProjectRoot}:/local openapitools/openapi-generator-cli generate -c /local/scripts/generator-cfg.yaml
    }
    "start" {
        try {
            mongo up --detach
            go run ${ProjectRoot}/cmd/inventory-api-service
        } finally {
            mongo down
        }
    }
    "test" {
        go test -v ./...
    }
    "docker" {
        docker build -t __docker-id__/inventory-webapi:local-build -f ${ProjectRoot}/build/docker/Dockerfile .
   }
    "mongo" {
        mongo up
    }
    default {
        throw "Unknown command: $command"
    }
}