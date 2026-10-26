# Commands

```bash
# Run all tests. The postgres store tests start a postgres:16-alpine container, so Docker must be running
make test
# or: go test -tags testing ./... | grep -v 'no test files'

# Run a single test
go test -tags testing ./internal/planet/internal/clicks/usecases/click_usecase/... -run TestName

# Run the concurrency-sensitive tests under the race detector
go test -tags testing ./... -race

# Fail on any unreachable function, production or test helper
make deadcode

# Lint. Needs golangci-lint on PATH at the pinned version: make setup-tools
make lint

# gofumpt every Go file and run go mod tidy — rewrites in place
make tidy

# Fail if `make tidy` would change anything, go.mod included
make check-format

# Run API server locally
go run ./cmd/api -config cmd/api/example.yaml

# Generate protobuf code (requires buf CLI)
make proto

# Refresh the embedded quiz bank from the shared /quiz, then commit it
make quiz

# Refresh the embedded tile coordinates blob from the shared /map, then commit it
make map

# Refresh the vendored VPN/datacenter ranges from upstream, then commit them
make vpn-lists

# Refresh the vendored disposable email domains from upstream, then commit them
make disposable-domains

# Refresh the vendored Zernio OpenAPI document from upstream, then regenerate the client from it; commit both
make zernio-spec
make zernio

# Build Docker image
make dBuild

# The clips job, as the clips workflow runs it (see clips-job.md)
go run ./cmd/clips fetch -ssh <user@host> -out ../frontend/replay.json
go run ./cmd/clips choose -shortlist ../frontend/shortlist.json
ZERNIO_API_KEY=... go run ./cmd/clips schedule -clips ../frontend/clips
# The model editor on a saved shortlist: -prompt prints what it would be sent and asks nothing (see clips-job.md, "The editor")
ANTHROPIC_API_KEY=... go run ./cmd/clips edit -shortlist ../frontend/shortlist.json [-prompt]
```
