package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pboyd/twig/api/gen/account/v1/accountv1connect"
	"github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	"github.com/pboyd/twig/api/gen/health/v1/healthv1connect"
	"github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	"github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/services/twig/internal/auth"
	"github.com/pboyd/twig/services/twig/internal/db"
	"github.com/pboyd/twig/services/twig/internal/handler"
)

type route struct {
	path string
	h    http.Handler
}

func services(queries *db.Queries, pool *pgxpool.Pool, limiter *auth.LoginLimiter) []route {
	var routes []route

	healthPath, healthH := healthv1connect.NewHealthServiceHandler(&handler.Health{Queries: queries})
	routes = append(routes, route{path: healthPath, h: healthH})

	taskPath, taskH := taskv1connect.NewTaskServiceHandler(&handler.Task{Queries: queries, Pool: pool})
	routes = append(routes, route{path: taskPath, h: taskH})

	planPath, planH := planv1connect.NewPlanServiceHandler(&handler.Plan{Queries: queries, Pool: pool})
	routes = append(routes, route{path: planPath, h: planH})

	goalPath, goalH := goalv1connect.NewGoalServiceHandler(&handler.Goal{Queries: queries, Pool: pool})
	routes = append(routes, route{path: goalPath, h: goalH})

	accountPath, accountH := accountv1connect.NewAccountServiceHandler(&handler.Account{Queries: queries, Limiter: limiter})
	routes = append(routes, route{path: accountPath, h: accountH})

	return routes
}

func newRouter(authQ auth.Querier, routes []route, cliBinary *handler.CLIBinary, webUI http.Handler, loginH, logoutH http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/auth/login", loginH)
	mux.Handle("/auth/logout", logoutH)

	authed := auth.Middleware(authQ)
	for _, rt := range routes {
		mux.Handle(rt.path, authed(rt.h))
	}

	cliMux := http.NewServeMux()
	cliMux.HandleFunc("GET /cli/download", cliBinary.ServeDownload)
	cliMux.HandleFunc("GET /cli/info", cliBinary.ServeInfo)
	mux.Handle("/cli/", authed(cliMux))

	mux.Handle("/", webUI)

	return mux
}
