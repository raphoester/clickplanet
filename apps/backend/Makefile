.PHONY: proto test dBuild dRun dRm dPush dLogin

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

dPush: dBuild 
	@docker tag clickplanet-back:local registry.digitalocean.com/clickplanet/backend:latest
	@docker push registry.digitalocean.com/clickplanet/backend:latest 

dLogin: 
	@doctl registry login