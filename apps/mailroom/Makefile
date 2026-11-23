.PHONY: build test vet deploy

build:
	go run github.com/syumai/workers-go/cmd/workers-assets-gen -mode=go
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o ./build/app.wasm .

test:
	go test ./...

vet:
	go vet ./...
	GOOS=js GOARCH=wasm go vet ./...

deploy: build
	npx wrangler@4 deploy
