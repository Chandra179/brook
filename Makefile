.PHONY: vendor swag mocks test lint ci run up down re fieldalignment modernize migrate-up migrate-down migrate-create

vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

mocks:
	go tool mockery

test:
	go test -short -race -count=1 ./...

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

rename:
	scripts/rename-module.sh example

fix:
	go fix ./...

align:
	@which fieldalignment >/dev/null 2>&1 || go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	fieldalignment -fix ./...

migrate-up:
	@which goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	goose -dir store/migrations postgres "$$POSTGRES_DSN" up

migrate-down:
	@which goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	goose -dir store/migrations postgres "$$POSTGRES_DSN" down

migrate-create:
	@which goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	goose -dir store/migrations create $(name) sql
