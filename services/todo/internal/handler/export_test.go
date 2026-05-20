package handler

import (
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/db"
)

func ExportDbTaskToProto(t db.Task) *taskv1.Task {
	return dbTaskToProto(t)
}
