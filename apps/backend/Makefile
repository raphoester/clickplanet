.PHONY: proto dbUp dbDown test migrateFromRedis

proto:
	@cd ../../proto && buf lint
	@cd proto && buf generate

# Redis is only needed for the tilesStorage.driver: redis path. The default
# memory driver needs nothing, and the integration tests use dockertest.
dbUp:
	@cd ./tools/devdb && \
		docker-compose up -d --wait
		docker exec -i clickPlanet-redis /bin/sh -c "(redis-cli -x script load < /static/setAndPublishOnStream.lua) > /static/setAndPublishOnStream.sha1"

dbDown:
	@cd ./tools/devdb && \
		docker-compose down

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

# One-off: copy tile state out of a running Redis into a snapshot file.
migrateFromRedis:
	@go run ./cmd/redis-to-snapshot -config cmd/api/example.yaml
