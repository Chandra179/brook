.PHONY: vendor swag test test-integration lint ci run up down re

vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

test:
	go test -short -race -count=1 ./...

test-integration:
	go test -tags=integration -race -count=1 -v ./...

lint:
	golangci-lint run

ci:
	act workflow_dispatch \
		--container-daemon-socket /var/run/docker.sock \
		--reuse

run:
	go run ./cmd/example/

up:
	docker compose up -d

down:
	docker compose down

re:
	scripts/rename-module.sh example
