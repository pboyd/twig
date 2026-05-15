.PHONY: proto build dev migrate-up migrate-down

proto:
	cd services/todo && buf generate

build:
	podman build -t todo-server services/todo

dev:
	podman-compose up --build

migrate-up:
	migrate -path services/todo/db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path services/todo/db/migrations -database "$(DATABASE_URL)" down 1
