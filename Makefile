.PHONY: test build-app build-operator fmt

test:
	go test ./...

build-app:
	go build -o bin/tiny-llm ./app/cmd/server

build-operator:
	go build -o bin/tiny-llm-operator ./operator/cmd/manager

fmt:
	go fmt ./...
