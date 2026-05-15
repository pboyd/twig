# Backend Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Stand up a wired Connect-Go server at `services/todo` with a single HealthService handler that runs `SELECT 1` against Postgres and returns `"ok"`.

**Architecture:** Connect-Go HTTP server (h2c for gRPC + HTTP/1.1 support); sqlc-generated DB access via pgx/v5; golang-migrate runs pending migrations on startup via the `database/postgres` driver; all services composed with podman-compose for dev.

**Tech Stack:** Go 1.24, connectrpc.com/connect, github.com/jackc/pgx/v5, github.com/sqlc-dev/sqlc, github.com/golang-migrate/migrate/v4, Buf CLI, Podman, podman-compose

---

### Task 1: Create directory structure and Go module

**Files:**
- Create: `services/todo/` (directory tree)
- Create: `services/todo/go.mod`

**Step 1: Create the directory tree**

```bash
cd /home/user/dev/todo
mkdir -p services/todo/{cmd/server,internal/{handler,db},db/{migrations,queries},proto/health/v1,gen}
```

**Step 2: Initialize the Go module**

```bash
cd services/todo
go mod init camelot.email/todo
```

> `camelot.email/todo` is the module path used throughout this plan. Change it to match your VCS path before pushing (e.g., `github.com/yourorg/todo`).

**Step 3: Verify**

```bash
cat go.mod
```

Expected: file exists, first line is `module camelot.email/todo`, second is `go 1.24`

**Step 4: Commit**

```bash
git add services/todo/go.mod
git commit -m "feat(server): init services/todo Go module"
```

---

### Task 2: Write proto definition and Buf config

**Files:**
- Create: `services/todo/proto/health/v1/health.proto`
- Create: `services/todo/buf.yaml`
- Create: `services/todo/buf.gen.yaml`

**Step 1: Write the proto file**

`services/todo/proto/health/v1/health.proto`:
```protobuf
syntax = "proto3";

package health.v1;

option go_package = "camelot.email/todo/gen/health/v1;healthv1";

service HealthService {
  rpc Check(CheckRequest) returns (CheckResponse);
}

message CheckRequest {}

message CheckResponse {
  string status = 1;
}
```

**Step 2: Write buf.yaml**

`services/todo/buf.yaml`:
```yaml
version: v2
modules:
  - path: proto
```

**Step 3: Write buf.gen.yaml**

`services/todo/buf.gen.yaml`:
```yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen
    opt:
      - paths=source_relative
  - remote: buf.build/connectrpc/go
    out: gen
    opt:
      - paths=source_relative
```

**Step 4: Commit**

```bash
git add services/todo/proto services/todo/buf.yaml services/todo/buf.gen.yaml
git commit -m "feat(server): add HealthService proto and Buf config"
```

---

### Task 3: Generate proto code

**Prerequisite:** `buf` CLI installed. Install with:
```bash
go install github.com/bufbuild/buf/cmd/buf@latest
```

**Step 1: Run buf generate**

```bash
cd services/todo
buf generate
```

Expected: no errors. `gen/health/v1/` directory created.

**Step 2: Verify generated files exist**

```bash
ls gen/health/v1/
ls gen/health/v1/healthv1connect/
```

Expected:
```
gen/health/v1/health.pb.go
gen/health/v1/healthv1connect/health.connect.go
```

The `health.pb.go` file contains the message types (`CheckRequest`, `CheckResponse`).
The `health.connect.go` file contains `NewHealthServiceHandler` and the `HealthServiceHandler` interface your handler must implement.

**Step 3: Commit**

```bash
git add services/todo/gen
git commit -m "feat(server): generate HealthService proto stubs"
```

---

### Task 4: Set up sqlc config, migration, and query

**Files:**
- Create: `services/todo/sqlc.yaml`
- Create: `services/todo/db/migrations/000001_init.up.sql`
- Create: `services/todo/db/migrations/000001_init.down.sql`
- Create: `services/todo/db/queries/health.sql`

**Step 1: Write sqlc.yaml**

`services/todo/sqlc.yaml`:
```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/migrations/"
    gen:
      go:
        package: "db"
        out: "internal/db"
        sql_package: "pgx/v5"
```

**Step 2: Write the initial migration (intentionally empty)**

`services/todo/db/migrations/000001_init.up.sql`:
```sql
-- Initial migration. Schema will be added in future features.
```

`services/todo/db/migrations/000001_init.down.sql`:
```sql
-- nothing to roll back
```

**Step 3: Write the health query**

`services/todo/db/queries/health.sql`:
```sql
-- name: Ping :one
SELECT 1::int AS result;
```

The `-- name: Ping :one` annotation tells sqlc to generate a `Ping(ctx) (int32, error)` method on `*Queries`.

**Step 4: Commit**

```bash
git add services/todo/sqlc.yaml services/todo/db
git commit -m "feat(server): add sqlc config, initial migration, health query"
```

---

### Task 5: Generate sqlc code

**Prerequisite:** `sqlc` CLI installed:
```bash
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

**Step 1: Run sqlc generate**

```bash
cd services/todo
sqlc generate
```

Expected: no errors. `internal/db/` populated.

**Step 2: Verify generated files**

```bash
ls internal/db/
```

Expected files: `db.go`, `health.sql.go`, `models.go`

- `db.go` — `DBTX` interface and `New(*pgxpool.Pool) *Queries`
- `health.sql.go` — `func (q *Queries) Ping(ctx context.Context) (int32, error)`
- `models.go` — empty (no tables yet)

**Step 3: Commit**

```bash
git add services/todo/internal/db
git commit -m "feat(server): generate sqlc DB layer"
```

---

### Task 6: Write the Connect-Go handler

**Files:**
- Create: `services/todo/internal/handler/health.go`

**Step 1: Write the handler**

`services/todo/internal/handler/health.go`:
```go
package handler

import (
	"context"

	"connectrpc.com/connect"

	healthv1 "camelot.email/todo/gen/health/v1"
	"camelot.email/todo/internal/db"
)

type Health struct {
	Queries *db.Queries
}

func (h *Health) Check(
	ctx context.Context,
	req *connect.Request[healthv1.CheckRequest],
) (*connect.Response[healthv1.CheckResponse], error) {
	if _, err := h.Queries.Ping(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&healthv1.CheckResponse{Status: "ok"}), nil
}
```

**Step 2: Commit**

```bash
git add services/todo/internal/handler
git commit -m "feat(server): add Health Connect handler"
```

---

### Task 7: Write main.go and resolve dependencies

**Files:**
- Create: `services/todo/cmd/server/main.go`
- Modify: `services/todo/go.mod` (via `go mod tidy`)
- Create: `services/todo/go.sum` (via `go mod tidy`)

**Step 1: Write main.go**

`services/todo/cmd/server/main.go`:
```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"camelot.email/todo/gen/health/v1/healthv1connect"
	"camelot.email/todo/internal/db"
	"camelot.email/todo/internal/handler"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		log.Fatalf("migrate new: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migrate up: %v", err)
	}
	log.Println("migrations applied")

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("pgxpool: %v", err)
	}
	defer pool.Close()

	queries := db.New(pool)

	mux := http.NewServeMux()
	path, h := healthv1connect.NewHealthServiceHandler(&handler.Health{Queries: queries})
	mux.Handle(path, h)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", h2c.NewHandler(mux, &http2.Server{})); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
```

> **h2c:** Connect-Go uses HTTP/2 for gRPC clients. `h2c` (HTTP/2 cleartext) handles this without TLS. The Connect HTTP protocol also works over HTTP/1.1 — the `h2c` wrapper serves both transparently.

**Step 2: Fetch dependencies and tidy**

```bash
cd services/todo
go get connectrpc.com/connect@latest
go get github.com/golang-migrate/migrate/v4@latest
go get github.com/jackc/pgx/v5@latest
go get golang.org/x/net@latest
go mod tidy
```

`go mod tidy` will pull in `lib/pq` and other transitive deps automatically via the `_ "github.com/golang-migrate/migrate/v4/database/postgres"` side-effect import.

**Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: no errors. (The binary will fail at runtime without a database — that's expected.)

**Step 4: Commit**

```bash
git add services/todo/cmd services/todo/go.mod services/todo/go.sum
git commit -m "feat(server): wire up Connect-Go server in main.go"
```

---

### Task 8: Write Dockerfile

**Files:**
- Create: `services/todo/Dockerfile`

**Step 1: Write Dockerfile**

`services/todo/Dockerfile`:
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=builder /server /server
COPY --from=builder /app/db/migrations /db/migrations
EXPOSE 8080
ENTRYPOINT ["/server"]
```

> The migration files are copied to `/db/migrations` in the final image. The server binary runs with working directory `/` (distroless default), so `file://db/migrations` resolves correctly to `/db/migrations`.

**Step 2: Commit**

```bash
git add services/todo/Dockerfile
git commit -m "feat(server): add multi-stage Dockerfile (distroless final)"
```

---

### Task 9: Write compose.yaml and Makefile

**Files:**
- Create: `services/todo/compose.yaml`
- Create: `services/todo/Makefile`

**Step 1: Write compose.yaml**

`services/todo/compose.yaml`:
```yaml
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: todo
      POSTGRES_PASSWORD: todo
      POSTGRES_DB: todo
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U todo"]
      interval: 5s
      timeout: 5s
      retries: 5
    ports:
      - "5432:5432"

  server:
    build: .
    environment:
      DATABASE_URL: postgres://todo:todo@postgres:5432/todo?sslmode=disable
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
```

**Step 2: Write Makefile**

`services/todo/Makefile`:
```makefile
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
```

> Makefile targets use tabs, not spaces. If your editor converts tabs to spaces, the make commands will fail with `missing separator`.

**Step 3: Commit**

```bash
git add services/todo/compose.yaml services/todo/Makefile
git commit -m "feat(server): add compose.yaml and Makefile"
```

---

### Task 10: Smoke test

**Prerequisites:** `podman` and `podman-compose` installed and daemon running.

**Step 1: Start the dev environment**

```bash
cd services/todo
make dev
```

Watch the logs. Expected sequence:
1. `postgres` container starts, passes healthcheck
2. `server` container builds and starts
3. Server logs: `migrations applied`
4. Server logs: `listening on :8080`

**Step 2: Send a health check (Connect HTTP protocol)**

In a new terminal:

```bash
curl -s -X POST http://localhost:8080/health.v1.HealthService/Check \
  -H "Content-Type: application/json" \
  -d '{}'
```

Expected response:
```json
{"status":"ok"}
```

**Step 3: Stop**

```bash
podman-compose down
```

**Step 4: Commit**

No new files — verification only. If the smoke test passed, the foundation is complete.

```bash
git tag foundation-smoke-test-passed
```

---

## Notes

- The `internal/db/` directory is entirely sqlc-generated. Do not edit it by hand — regenerate with `sqlc generate`.
- The `gen/` directory is entirely buf-generated. Do not edit it by hand — regenerate with `make proto` or `buf generate`.
- The `golang-migrate` library handles the `file://` URL by treating the path after `://` as relative to the process working directory. This is why the migrations path in `main.go` matches the copy destination in `Dockerfile`.
- When adding real schema in future features, add new numbered migration files (`000002_...`) — never modify `000001_init`.
