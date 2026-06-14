vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

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
