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

	"github.com/pboyd/todo/services/todo/gen/health/v1/healthv1connect"
	"github.com/pboyd/todo/services/todo/internal/db"
	"github.com/pboyd/todo/services/todo/internal/handler"
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
