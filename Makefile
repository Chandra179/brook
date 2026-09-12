LEFTHOOK_VERSION ?= v2.1.9
LEFTHOOK_BIN ?= $(CURDIR)/bin/lefthook

.PHONY: vendor swag mocks format-check test lint vet check ci run re fieldalignment modernize migrate-up migrate-down migrate-create lefthook-install lefthook

# Load env vars from .env and export them to all recipe shells
-include .env
export

vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

mocks:
	go run github.com/vektra/mockery/v3

format-check:
	@test -z "$$(find . -path './vendor' -prune -o -path './.git' -prune -o -type f -name '*.go' -print0 | xargs -0 gofmt -l)" || (echo "Go files are not formatted; run gofmt -w" && exit 1)

test:
	go test -short -race -count=1 ./...

lint:
	golangci-lint run

vet:
	go vet ./...

check: format-check lint vet test

lefthook-install:
	@mkdir -p "$(CURDIR)/bin"
	@if [ ! -x "$(LEFTHOOK_BIN)" ]; then GOBIN="$(CURDIR)/bin" go install github.com/evilmartians/lefthook/v2@$(LEFTHOOK_VERSION); fi

lefthook: lefthook-install
	"$(LEFTHOOK_BIN)" install

ci:
	act workflow_dispatch \
		--container-daemon-socket /var/run/docker.sock \
		--reuse

run:
	@fuser -k 8080/tcp >/dev/null 2>&1 || true
	go run ./cmd/example/

rename:
	scripts/rename-module.sh $(name)

fix:
	go fix ./...

align:
	@which fieldalignment >/dev/null 2>&1 || go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	fieldalignment -fix ./...

modernize:
	go fix ./...

migrate-up:
	@which goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	@test -n "$$SQLITE_DSN" || (echo "SQLITE_DSN must be set" && exit 1)
	goose -dir store/migrations/sqlite sqlite3 "$$SQLITE_DSN" up

migrate-down:
	@which goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	@test -n "$$SQLITE_DSN" || (echo "SQLITE_DSN must be set" && exit 1)
	goose -dir store/migrations/sqlite sqlite3 "$$SQLITE_DSN" down

migrate-create:
	@which goose >/dev/null 2>&1 || go install github.com/pressly/goose/v3/cmd/goose@latest
	goose -dir store/migrations/sqlite create $(name) sql
