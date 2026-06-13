vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

ci:
	go build ./...
	go vet ./...
	go mod tidy && go mod vendor && git diff --exit-code
	go install github.com/swaggo/swag/cmd/swag@v1.16.4
	swag init -g cmd/example/main.go -o docs
	git diff --exit-code docs/
	docker build .

run:
	go run ./cmd/example/

up:
	docker compose up -d

down:
	docker compose down

re:
	scripts/rename-module.sh example
