package tui

import (
	"context"
	"net/http"
	"os"

	"connectrpc.com/connect"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

const defaultAddr = "http://localhost:8080"

// NewClient returns a TaskServiceClient using TODO_API_KEY and TODO_ADDR env vars.
func NewClient() (taskv1connect.TaskServiceClient, string) {
	apiKey := os.Getenv("TODO_API_KEY")
	addr := os.Getenv("TODO_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	client := taskv1connect.NewTaskServiceClient(
		&http.Client{},
		addr,
		connect.WithSendGzip(),
		connect.WithInterceptors(bearerInterceptor(apiKey)),
	)
	return client, addr
}

func bearerInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+apiKey)
			return next(ctx, req)
		})
	})
}
