.PHONY: proto test dBuild dRun dRm vpn-lists

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

test:
	@go test -v ./... | grep -v 'no test files'

# Refreshes the vendored VPN and datacenter ranges from X4BNet/lists_vpn, which
# rebuilds them daily from ASN ownership. They are checked in and embedded
# rather than fetched at boot, so refreshing them is a commit: run this, run the
# tests (they assert the lists still parse and are not truncated), and commit.
VPN_LISTS_URL := https://raw.githubusercontent.com/X4BNet/lists_vpn/main/output
VPN_LISTS_DIR := internal/kernel/ipblock/data

vpn-lists:
	@curl -sSf $(VPN_LISTS_URL)/vpn/ipv4.txt        -o $(VPN_LISTS_DIR)/vpn_ipv4.txt
	@curl -sSf $(VPN_LISTS_URL)/vpn/ipv6.txt        -o $(VPN_LISTS_DIR)/vpn_ipv6.txt
	@curl -sSf $(VPN_LISTS_URL)/datacenter/ipv4.txt -o $(VPN_LISTS_DIR)/datacenter_ipv4.txt
	@curl -sSf $(VPN_LISTS_URL)/datacenter/ipv6.txt -o $(VPN_LISTS_DIR)/datacenter_ipv6.txt
	@wc -l $(VPN_LISTS_DIR)/*.txt