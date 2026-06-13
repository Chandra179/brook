vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

run:
	go run ./cmd/example/

up:
	docker compose up -d

down:
	docker compose down

re:
	scripts/rename-module.sh example
