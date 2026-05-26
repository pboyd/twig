.PHONY: proto build cli dev migrate-up migrate-down deploy

INVENTORY ?= deploy/inventories/test.ini

proto:
	cd services/todo && buf generate

build:
	podman build -t todo-server services/todo

cli:
	cd services/todo && go build -o todo ./cmd/todo

dev:
	podman-compose up -d --build --force-recreate server

migrate-up:
	migrate -path services/todo/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path services/todo/db/migrations -database "$(DATABASE_URL)" down 1

deploy:
	ansible-playbook -i $(INVENTORY) deploy/site.yml
