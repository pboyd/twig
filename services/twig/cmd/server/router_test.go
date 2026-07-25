package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
)

type stubQuerier struct{}

func (stubQuerier) GetApiKeyByHash(_ context.Context, _ string) (db.ApiKey, error) {
	return db.ApiKey{}, errNotFound
}
func (stubQuerier) GetSession(_ context.Context, _ string) (db.Session, error) {
	return db.Session{}, errNotFound
}

var errNotFound = fmt.Errorf("not found")

func makeRouterFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>index</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('app')"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRouter_UnauthenticatedServicePathsReturn401(t *testing.T) {
	routes := services(nil, nil, nil)
	cliBinary := handler.NewCLIBinary(t.TempDir())
	webUI := handler.NewWebUI(makeRouterFixture(t))
	r := newRouter(stubQuerier{}, routes, cliBinary, webUI,
		http.NotFoundHandler(), http.NotFoundHandler())

	paths := []string{
		"/health.v1.HealthService/Check",
		"/task.v1.TaskService/ListTasks",
		"/plan.v1.PlanService/GetDailyPlan",
		"/goal.v1.GoalService/ListGoals",
		"/account.v1.AccountService/GetProfile",
		"/cli/download",
		"/cli/info",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s: want 401, got %d", p, rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); strings.Contains(ct, "text/html") {
				t.Errorf("%s: Content-Type should not be text/html, got %s", p, ct)
			}
		})
	}
}

func TestRouter_SPARoutesReturnIndex(t *testing.T) {
	dir := makeRouterFixture(t)
	routes := services(nil, nil, nil)
	cliBinary := handler.NewCLIBinary(t.TempDir())
	webUI := handler.NewWebUI(dir)
	r := newRouter(stubQuerier{}, routes, cliBinary, webUI,
		http.NotFoundHandler(), http.NotFoundHandler())

	for _, p := range []string{"/", "/tasks", "/login"} {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("%s: want 200, got %d", p, rec.Code)
			}
			if body := rec.Body.String(); body != "<html>index</html>" {
				t.Errorf("%s: body = %q, want index.html content", p, body)
			}
		})
	}
}

func TestRouter_AuthLoginReachable(t *testing.T) {
	routes := services(nil, nil, nil)
	cliBinary := handler.NewCLIBinary(t.TempDir())
	webUI := handler.NewWebUI(makeRouterFixture(t))
	loginH := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("login page"))
	})
	r := newRouter(stubQuerier{}, routes, cliBinary, webUI,
		loginH, http.NotFoundHandler())

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("/auth/login: want 200, got %d", rec.Code)
	}
}
