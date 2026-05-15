.PHONY: proto build dev migrate-up migrate-down

proto:
	buf generate

build:
	podman build -t todo-server .

dev:
	podman-compose up --build

migrate-up:
	migrate -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path db/migrations -database "$(DATABASE_URL)" down 1
