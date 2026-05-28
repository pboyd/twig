package tui

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/config"
)

const defaultAddr = "http://localhost:8080"

// NewClient returns a TaskServiceClient using the resolved Config values.
func NewClient(cfg config.Config) (taskv1connect.TaskServiceClient, string) {
	client := taskv1connect.NewTaskServiceClient(
		&http.Client{},
		cfg.APIURL,
		connect.WithSendGzip(),
		connect.WithInterceptors(bearerInterceptor(cfg.APIKey)),
	)
	return client, cfg.APIURL
}

func bearerInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+apiKey)
			return next(ctx, req)
		})
	})
}
