package handler

import (
	"context"

	"connectrpc.com/connect"

	healthv1 "github.com/pboyd/twig/api/gen/health/v1"
	"github.com/pboyd/twig/services/twig/internal/db"
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
