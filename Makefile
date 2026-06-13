vendor:
	go mod tidy && go mod vendor

swag:
	swag init -g cmd/example/main.go -o docs

up:
	docker compose up -d

re:
	scripts/rename-module.sh example