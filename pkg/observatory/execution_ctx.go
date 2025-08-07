package observatory

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/simple-container-com/go-aws-lambda-sdk/pkg/logger"
)

func WithExecutionCtx(l logger.Logger, ctx context.Context, req http.Request) context.Context {
	executionCtxJson := req.Header.Get("X-Everworker-Execution-Context")
	if executionCtxJson == "" {
		return ctx
	}
	var executionCtx map[string]any
	if err := json.Unmarshal([]byte(executionCtxJson), &executionCtx); err != nil {
		l.Errorf(ctx, "Failed to unmarshal execution context: %v", err)
		return ctx
	}
	return l.WithValue(ctx, "executionCtx", executionCtx)
}
