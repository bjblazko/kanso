# Hüpattl! Kanso

# Run locally with dev login and throwaway data under ./tmp
run:
    mkdir -p tmp/data tmp/state
    KANSO_DATA_DIR=tmp/data KANSO_STATE_DIR=tmp/state KANSO_BASE_URL=http://localhost:8090 KANSO_DEV_LOGIN=true go run ./cmd/kanso

test:
    go test ./...

lint:
    gofmt -l .
    go vet ./...

# Refresh vendored frontend assets (pinned versions, checksums in the scripts)
vendor:
    ./scripts/vendor-plex.sh
    ./scripts/vendor-mermaid.sh
