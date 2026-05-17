vendor:
	go mod tidy && go mod vendor

up:
	docker compose up -d

re:
	scripts/rename-module.sh example