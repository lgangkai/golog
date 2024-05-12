package golog

import (
	"context"
	"testing"
	"time"
)

func TestFileCreateAndDelete(t *testing.T) {
	logger := Default()
	logger.maxKeepDays = 3
	logger.date = int32(time.Now().YearDay() - 1)

	createDateFolder(logger.rootPath, "2024-05-12")
	createDateFolder(logger.rootPath, "2024-05-11")
	createDateFolder(logger.rootPath, "2024-05-10")
	createDateFolder(logger.rootPath, "2024-05-09")

	logger.Debug(context.Background(), "test logger")
	logger.Debug(context.Background(), "test logger")
	logger.Debug(context.Background(), "test logger")
}

func TestLogTrack(t *testing.T) {
	logger := Default()
	ctx := context.Background()
	ctx = context.WithValue(ctx, "traceKey", map[string]any{
		"request_id": 123,
		"user_id":    "456",
	})
	logger.Debug(ctx, "log for debug")
	logger.Info(ctx, "log for info")
	logger.Warning(ctx, "log for warning")
	logger.Error(ctx, "log for error")
}
