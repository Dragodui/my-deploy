.PHONY: build-cli build-agent build-server build

build: build-cli build-agent build-server

build-cli:
	go build -o bin/mydeploy ./cmd/cli

build-agent:
	go build -o bin/mydeploy-agent ./cmd/agent

build-server:
	go build -o bin/mydeploy-server ./cmd
