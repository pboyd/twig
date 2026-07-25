package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
)

// provisionFlag is a repeatable --provision-user=name:password flag.
type provisionFlag []string

func (f *provisionFlag) String() string { return strings.Join(*f, ", ") }
func (f *provisionFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func main() {
	var provisions provisionFlag
	flag.Var(&provisions, "provision-user", "provision a user account: name:password (repeatable)")
	flag.Parse()

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

	if len(provisions) > 0 {
		for _, spec := range provisions {
			idx := strings.Index(spec, ":")
			if idx < 1 {
				log.Fatalf("--provision-user: expected name:password, got %q", spec)
			}
			name := spec[:idx]
			password := spec[idx+1:]
			rawKey, err := auth.ProvisionUser(context.Background(), queries, name, password)
			if err != nil {
				log.Fatalf("provision user %q: %v", name, err)
			}
			fmt.Printf("provisioned user %q — API key: %s\n", name, rawKey)
		}
		return
	}

	const sessionLifetime = 30 * 24 * time.Hour

	// Rate-limit: block an IP or username after 5 failed logins within 15 minutes.
	const (
		loginMaxFailures = 5
		loginWindow      = 15 * time.Minute
	)
	loginLimiter := auth.NewLoginLimiter(loginMaxFailures, loginWindow)

	cliBinary := handler.NewCLIBinary("/cli")
	webUI := handler.NewWebUI("/web")

	routes := services(queries, pool, loginLimiter)
	mux := newRouter(queries, routes, cliBinary, webUI,
		auth.LoginHandler(queries, sessionLifetime, loginLimiter),
		auth.LogoutHandler(queries),
	)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", h2c.NewHandler(mux, &http2.Server{})); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
