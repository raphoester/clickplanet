.PHONY: proto map quiz test lint lint-ci tidy check-format check-quiz setup-tools deadcode dBuild dRun dRm vpn-lists disposable-domains metrics

TEST_TAG := testing

proto:
	@cd ../../proto && buf lint
	@cd proto && buf generate

MAP_DIR := ../../map
MAP_ASSET_DIR := generated/map

map:
	@for asset in coordinates borders; do \
		test -n "$$(ls $(MAP_DIR)/$$asset-*.bin 2>/dev/null)" || { echo "map: no $(MAP_DIR)/$$asset-<hash>.bin" >&2; exit 1; }; \
		rm -f $(MAP_ASSET_DIR)/$$asset-*.bin; \
		cp $(MAP_DIR)/$$asset-*.bin $(MAP_ASSET_DIR)/; \
		ls $(MAP_ASSET_DIR)/$$asset-*.bin; \
	done

QUIZ_DIR := ../../quiz
QUIZ_ASSET_DIR := generated/quiz

quiz:
	@test -f $(QUIZ_DIR)/bank.json || { echo "quiz: no $(QUIZ_DIR)/bank.json" >&2; exit 1; }
	@mkdir -p $(QUIZ_ASSET_DIR)
	@cp $(QUIZ_DIR)/bank.json $(QUIZ_ASSET_DIR)/
	@ls $(QUIZ_ASSET_DIR)/bank.json

check-quiz:
	@cmp -s $(QUIZ_DIR)/bank.json $(QUIZ_ASSET_DIR)/bank.json || { \
		echo "$(QUIZ_ASSET_DIR)/bank.json is not $(QUIZ_DIR)/bank.json. Run 'make quiz'." >&2; \
		exit 1; \
	}

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

# bash -o pipefail, not .SHELLFLAGS: macOS ships GNU Make 3.81.
test:
	@bash -o pipefail -c "go test -v -tags $(TEST_TAG) ./... | grep -v 'no test files'"

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

GOLANGCI_LINT_VERSION := v2.13.1

lint:
	@golangci-lint run --timeout=10m

lint-ci:
	@golangci-lint run --timeout=10m --allow-parallel-runners

tidy:
	@go mod tidy
	@golangci-lint fmt

# golangci-lint fmt --diff exits 0 either way, so test its output.
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

setup-tools:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
		| sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

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

DISPOSABLE_DOMAINS_URL := https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/main/disposable_email_blocklist.conf
DISPOSABLE_DOMAINS_FILE := internal/auth/internal/signin/embedded_disposable_domains/disposable_email_blocklist.conf

disposable-domains:
	@curl -sSf $(DISPOSABLE_DOMAINS_URL) -o $(DISPOSABLE_DOMAINS_FILE)
	@wc -l $(DISPOSABLE_DOMAINS_FILE)

METRIC ?= .

metrics:
	@docker compose -f ../../deploy/vps/docker-compose.yaml exec -T backend \
		wget -qO- localhost:8080/metrics | grep -E '$(METRIC)'
