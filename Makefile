.PHONY: vendor swag mocks test test-integration lint ci run up down re fieldalignment modernize profiler dashboards

vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

mocks:
	go tool mockery

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

modernize:
	go fix ./...

align:
	@which fieldalignment >/dev/null 2>&1 || go install golang.org/x/tools/go/analysis/passes/fieldalignment/cmd/fieldalignment@latest
	fieldalignment -fix ./...

profiler:
	docker compose up -d pyroscope grafana

dashboards:
	docker compose up -d pyroscope grafana
