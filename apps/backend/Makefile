.PHONY: proto map quiz test lint lint-ci tidy check-format setup-tools deadcode dBuild dRun dRm vpn-lists metrics

# Test helpers that more than one package needs cannot live in a _test.go file,
# so they live in an ordinary .go file carrying this tag. The tag is what keeps
# them out of the binary — a naming convention would not — and it is what lets
# `make deadcode` tell a helper apart from production code without matching on
# filenames.
TEST_TAG := testing

proto:
	@cd ../../proto && buf lint
	@cd proto && buf generate

# The tile coordinates and borders blobs, the same way `proto` works: the monorepo-shared /map is the
# source and generated/ holds this app's copy, because its Docker build context is its own directory.
# The copy is embedded, so run this and commit. The name is the content hash, so the previous one has
# to go or the embed glob matches two files and refuses to pick.
MAP_DIR := ../../map
MAP_ASSET_DIR := generated/map

map:
	@for asset in coordinates borders; do \
		test -n "$$(ls $(MAP_DIR)/$$asset-*.bin 2>/dev/null)" || { echo "map: no $(MAP_DIR)/$$asset-<hash>.bin" >&2; exit 1; }; \
		rm -f $(MAP_ASSET_DIR)/$$asset-*.bin; \
		cp $(MAP_DIR)/$$asset-*.bin $(MAP_ASSET_DIR)/; \
		ls $(MAP_ASSET_DIR)/$$asset-*.bin; \
	done

# The quiz bank, the same way `map` works, and for the same reason. The frontend deliberately has no
# copy: the answers are in it. See /quiz/README.md.
QUIZ_DIR := ../../quiz
QUIZ_ASSET_DIR := generated/quiz

quiz:
	@test -n "$$(ls $(QUIZ_DIR)/bank-*.json 2>/dev/null)" || { echo "quiz: no $(QUIZ_DIR)/bank-<hash>.json" >&2; exit 1; }
	@mkdir -p $(QUIZ_ASSET_DIR)
	@rm -f $(QUIZ_ASSET_DIR)/bank-*.json
	@cp $(QUIZ_DIR)/bank-*.json $(QUIZ_ASSET_DIR)/
	@ls $(QUIZ_ASSET_DIR)/bank-*.json

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
# here, as .custom-gcl.yml must. Install it with `make setup-tools`.
GOLANGCI_LINT_VERSION := v2.13.1

# The linters run through ./custom-gcl: the pinned golangci-lint rebuilt with the
# module plugins of .custom-gcl.yml compiled in, because the stock binary cannot
# load them. Rebuilt whenever that file changes. The stock binary still builds it
# and runs the formatters.
custom-gcl: .custom-gcl.yml
	@golangci-lint custom

lint: custom-gcl
	@./custom-gcl run --timeout=10m

# --allow-parallel-runners so the pre-push hook can run this beside the other
# checks without the two fighting over the lint cache.
lint-ci: custom-gcl
	@./custom-gcl run --timeout=10m --allow-parallel-runners

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
#
# X4BNet works from ASN ownership, so it misses VPN servers rented inside other
# networks. Four more sources fill that in, all folded into the vpn list:
#   - Joe12387/open-source-vpn-ip-lists (CC0): each provider's own server list
#   - az0/vpn_ip (GPL-3): VPN hostnames resolved, from apps and extensions
#   - the Tor Project's exit list
#   - every range registered under a netname in VPN_NETNAMES, from the RIPE and
#     ARIN RDAP servers. Firefox's built-in VPN is announced by Fastly, so no
#     ASN list catches it; its registration name does.
# Needs jq. Bare addresses become /32 and /128, since the parser takes prefixes.
VPN_LISTS_URL := https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output
VPN_LISTS_DIR := internal/shared/cpipblock/cpdata
VPN_PROVIDERS_API := https://api.github.com/repos/Joe12387/open-source-vpn-ip-lists/contents/
VPN_AZ0_URL := https://az0-vpnip-public.oooninja.com/ip.txt
TOR_EXITS_URL := https://check.torproject.org/torbulkexitlist
VPN_NETNAMES := MOZILLA-FIREFOX-VPN
VPN_RDAP_SERVERS := https://rdap.db.ripe.net https://rdap.arin.net/registry
TO_PREFIX := awk '$$1 ~ /^[0-9a-fA-F:.]+$$/ { print $$1 (index($$1, ":") ? "/128" : "/32") }'

vpn-lists:
	@command -v jq >/dev/null || { echo "vpn-lists needs jq" >&2; exit 1; }
	@curl -sSf $(VPN_LISTS_URL)/vpn/ipv4.txt        -o $(VPN_LISTS_DIR)/vpn_ipv4.txt
	@curl -sSf $(VPN_LISTS_URL)/vpn/ipv6.txt        -o $(VPN_LISTS_DIR)/vpn_ipv6.txt
	@curl -sSf $(VPN_LISTS_URL)/datacenter/ipv4.txt -o $(VPN_LISTS_DIR)/datacenter_ipv4.txt
	@curl -sSf $(VPN_LISTS_URL)/datacenter/ipv6.txt -o $(VPN_LISTS_DIR)/datacenter_ipv6.txt
	@set -e; tmp=$$(mktemp -d); trap 'rm -rf "$$tmp"' EXIT; \
		curl -sSf $(VPN_PROVIDERS_API) -o "$$tmp/index.json"; \
		for u in $$(jq -r '.[] | select(.name | endswith(".txt")) | .download_url' "$$tmp/index.json"); do \
			curl -sSf "$$u" >> "$$tmp/providers"; \
		done; \
		$(TO_PREFIX) "$$tmp/providers" > $(VPN_LISTS_DIR)/vpn_providers.txt; \
		curl -sSf $(VPN_AZ0_URL) -o "$$tmp/az0"; \
		$(TO_PREFIX) "$$tmp/az0" > $(VPN_LISTS_DIR)/vpn_az0.txt; \
		curl -sSf $(TOR_EXITS_URL) -o "$$tmp/tor"; \
		$(TO_PREFIX) "$$tmp/tor" > $(VPN_LISTS_DIR)/tor_exits.txt; \
		: > "$$tmp/netnames"; \
		for name in $(VPN_NETNAMES); do for server in $(VPN_RDAP_SERVERS); do \
			curl -sSf "$$server/ips?name=$$name" -o "$$tmp/rdap.json"; \
			jq -r '.ipSearchResults[]?.cidr0_cidrs[]? | "\(.v4prefix // .v6prefix)/\(.length)"' "$$tmp/rdap.json" >> "$$tmp/netnames"; \
		done; done; \
		mv "$$tmp/netnames" $(VPN_LISTS_DIR)/vpn_netnames.txt
	@wc -l $(VPN_LISTS_DIR)/*.txt

# Reads the metrics endpoint on the droplet. It is not routed through Caddy (see
# the commented handle in deploy/vps/caddy/Caddyfile), so this goes through compose.
# METRIC filters: `make metrics METRIC=click_reaction`.
METRIC ?= .

metrics:
	@docker compose -f ../../deploy/vps/docker-compose.yaml exec -T backend \
		wget -qO- localhost:8080/metrics | grep -E '$(METRIC)'
