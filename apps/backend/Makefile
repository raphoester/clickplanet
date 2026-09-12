.PHONY: proto test lint lint-ci tidy check-format setup-tools deadcode dBuild dRun dRm vpn-lists metrics

# Test helpers that more than one package needs cannot live in a _test.go file,
# so they live in an ordinary .go file carrying this tag. The tag is what keeps
# them out of the binary — a naming convention would not — and it is what lets
# `make deadcode` tell a helper apart from production code without matching on
# filenames.
TEST_TAG := testing

proto:
	@cd ../../proto && buf lint
	@cd proto && buf generate

dBuild:
	@docker build -t clickplanet-back:local .

dRun:
	@docker run \
		-d \
		-p 8080:8080 \
		--name clickplanet \
		-v ./cmd/api:/home/app/config \
		clickplanet-back:local \
		/home/app/api -config /home/app/config/config.yaml

dRm:
	@docker rm -f clickplanet

# Run under bash for pipefail: without it the recipe takes grep's exit status,
# so a failing suite would report success and CI would believe it. `.SHELLFLAGS`
# would be the tidier way to say this, but it needs GNU Make 3.82 and macOS
# ships 3.81.
test:
	@bash -o pipefail -c "go test -v -tags $(TEST_TAG) ./... | grep -v 'no test files'"

# Dead code is a failure here, not a report. Two passes, because "is this
# reachable?" has two different right answers depending on whether test code
# counts as a caller:
#
#   production — no tests, no $(TEST_TAG) files. Everything left has to be
#     reachable from a main package, so a function only the tests call is
#     reported. That is the case a plain `deadcode -test` misses.
#   helpers — test binaries and $(TEST_TAG) files. Only findings inside the
#     tagged files are kept; production was already judged by the first pass,
#     and judging it again here would forgive exactly what that pass caught.
#
# deadcode is fetched at a pinned version rather than expected on PATH, so this
# needs no setup step and gives the same answer on every machine.
DEADCODE := go run golang.org/x/tools/cmd/deadcode@v0.50.0

deadcode:
	@tagged=$$(grep -rl --include='*.go' '^//go:build .*$(TEST_TAG)' . | sed 's|^\./||'); \
	found=$$($(DEADCODE) ./...); \
	if [ -n "$$tagged" ]; then \
		with_tests=$$($(DEADCODE) -test -tags $(TEST_TAG) ./...); \
		for f in $$tagged; do \
			found=$$(printf '%s\n%s' "$$found" "$$(printf '%s\n' "$$with_tests" | grep "^$$f:" || true)"); \
		done; \
	fi; \
	found=$$(printf '%s\n' "$$found" | grep -v '^$$' || true); \
	if [ -n "$$found" ]; then \
		printf '%s\n' "$$found"; \
		echo; \
		echo 'dead code: delete it, or give it a caller.'; \
		exit 1; \
	fi

# golangci-lint is pinned: an unpinned linter turns a green branch red on
# somebody else's release schedule, and the CI job reads the same version from
# here. Install it with `make setup-tools`.
GOLANGCI_LINT_VERSION := v2.13.1

lint:
	@golangci-lint run --timeout=10m

# --allow-parallel-runners so the pre-push hook can run this beside the other
# checks without the two fighting over the lint cache.
lint-ci:
	@golangci-lint run --timeout=10m --allow-parallel-runners

# gofumpt runs through golangci-lint's formatters block rather than as its own
# binary: one pinned version to install instead of two that can disagree, and it
# already knows which files are generated.
#
# Mutates the tree — `go mod tidy` rewrites go.mod/go.sum in place.
tidy:
	@go mod tidy
	@golangci-lint fmt

# `golangci-lint fmt --diff` prints the diff but exits 0 either way, so the
# output is what has to be tested rather than the status.
check-format:
	@out=$$(golangci-lint fmt --diff 2>&1); \
	if [ -n "$$out" ]; then \
		echo "These files are not formatted. Run 'make tidy':"; \
		printf '%s\n' "$$out" | grep '^diff ' | awk '{print "  " $$3}'; \
		exit 1; \
	fi
	@go mod tidy
	@git diff --quiet -- go.mod go.sum || { \
		echo "go.mod or go.sum is not tidy. Run 'make tidy' and commit the result."; \
		git --no-pager diff --stat -- go.mod go.sum; \
		exit 1; }

setup-tools: ## Install the pinned linter into $(go env GOPATH)/bin
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
		| sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

# Refreshes the vendored VPN and datacenter ranges from X4BNet/lists_vpn, which
# rebuilds them daily from ASN ownership. They are checked in and embedded
# rather than fetched at boot, so refreshing them is a commit: run this, run the
# tests (they assert the lists still parse and are not truncated), and commit.
VPN_LISTS_URL := https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output
VPN_LISTS_DIR := internal/shared/cpipblock/cpdata

vpn-lists:
	@curl -sSf $(VPN_LISTS_URL)/vpn/ipv4.txt        -o $(VPN_LISTS_DIR)/vpn_ipv4.txt
	@curl -sSf $(VPN_LISTS_URL)/vpn/ipv6.txt        -o $(VPN_LISTS_DIR)/vpn_ipv6.txt
	@curl -sSf $(VPN_LISTS_URL)/datacenter/ipv4.txt -o $(VPN_LISTS_DIR)/datacenter_ipv4.txt
	@curl -sSf $(VPN_LISTS_URL)/datacenter/ipv6.txt -o $(VPN_LISTS_DIR)/datacenter_ipv6.txt
	@wc -l $(VPN_LISTS_DIR)/*.txt

# Reads the metrics endpoint on the droplet. It is not routed through Caddy (see
# the commented handle in deploy/vps/Caddyfile), so this goes through compose.
# METRIC filters: `make metrics METRIC=click_reaction`.
METRIC ?= .

metrics:
	@docker compose -f ../../deploy/vps/docker-compose.yaml exec -T backend \
		wget -qO- localhost:8080/metrics | grep -E '$(METRIC)'
