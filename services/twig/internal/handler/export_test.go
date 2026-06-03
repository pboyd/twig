package handler

import (
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/services/twig/internal/db"
)

func ExportDbTaskToProto(t db.Task) *taskv1.Task {
	return dbTaskToProto(t)
}
