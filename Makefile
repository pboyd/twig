.PHONY: proto build cli dev migrate-up migrate-down deploy

INVENTORY ?= deploy/inventories/test.ini

proto:
	cd api && buf generate

build:
	podman build -t twig-server -f services/twig/Dockerfile .

cli:
	go build -o twig ./cmd/twig

cli-install:
	go install ./cmd/twig

dev:
	podman-compose up -d --build --force-recreate server

migrate-up:
	migrate -path services/twig/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path services/twig/db/migrations -database "$(DATABASE_URL)" down 1

deploy:
	ansible-playbook -J -i $(INVENTORY) deploy/site.yml
