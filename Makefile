.PHONY: proto build cli dev migrate-up migrate-down deploy

INVENTORY ?= deploy/inventories/test.ini

proto:
	cd services/twig && buf generate

build:
	podman build -t twig-server services/twig

cli:
	cd services/twig && go build -o twig ./cmd/twig

dev:
	podman-compose up -d --build --force-recreate server

migrate-up:
	migrate -path services/twig/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path services/twig/db/migrations -database "$(DATABASE_URL)" down 1

deploy:
	ansible-playbook -i $(INVENTORY) deploy/site.yml
