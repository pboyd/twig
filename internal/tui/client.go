package tui

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/config"
)

const defaultAddr = "http://localhost:8080"

// NewClient returns a TaskServiceClient, PlanServiceClient, GoalServiceClient,
// and the resolved API URL.
func NewClient(cfg config.Config) (taskv1connect.TaskServiceClient, planv1connect.PlanServiceClient, goalv1connect.GoalServiceClient, string) {
	httpClient := &http.Client{}
	opts := []connect.ClientOption{
		connect.WithSendGzip(),
		connect.WithInterceptors(bearerInterceptor(cfg.APIKey)),
	}
	taskClient := taskv1connect.NewTaskServiceClient(httpClient, cfg.APIURL, opts...)
	planClient := planv1connect.NewPlanServiceClient(httpClient, cfg.APIURL, opts...)
	goalClient := goalv1connect.NewGoalServiceClient(httpClient, cfg.APIURL, opts...)
	return taskClient, planClient, goalClient, cfg.APIURL
}

func bearerInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+apiKey)
			return next(ctx, req)
		})
	})
}
