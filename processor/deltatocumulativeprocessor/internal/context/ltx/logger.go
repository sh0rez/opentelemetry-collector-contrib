package ltx

import (
	"context"

	"go.uber.org/zap"
)

type key struct{}

func From(ctx context.Context) *zap.Logger {
	return ctx.Value(key{}).(*zap.Logger)
}

func With(ctx context.Context, log *zap.Logger) context.Context {
	return context.WithValue(ctx, key{}, log)
}
