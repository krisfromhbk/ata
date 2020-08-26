// Package zapadapter provides a logger that writes to a go.uber.org/zap.Logger.
package zapadapter

import (
	"context"

	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type key string

var idKey key

type Logger struct {
	logger *zap.Logger
}

func NewContextWithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, idKey, id)
}

func IDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(idKey).(string)
	return id, ok
}

func NewLogger(logger *zap.Logger) *Logger {
	return &Logger{logger: logger.WithOptions(zap.AddCallerSkip(1))}
}

func (pl *Logger) Log(ctx context.Context, level pgx.LogLevel, msg string, data map[string]interface{}) {
	id, ok := IDFromContext(ctx)

	fields := make([]zapcore.Field, len(data))
	i := 0
	for k, v := range data {
		fields[i] = zap.Reflect(k, v)
		i++
	}

	if ok {
		fields = append([]zapcore.Field{zap.String("request_id", id)}, fields...)
	}

	switch level {
	case pgx.LogLevelTrace:
		pl.logger.Debug(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	case pgx.LogLevelDebug:
		pl.logger.Debug(msg, fields...)
	case pgx.LogLevelInfo:
		pl.logger.Info(msg, fields...)
	case pgx.LogLevelWarn:
		pl.logger.Warn(msg, fields...)
	case pgx.LogLevelError:
		pl.logger.Error(msg, fields...)
	default:
		pl.logger.Error(msg, append(fields, zap.Stringer("PGX_LOG_LEVEL", level))...)
	}
}
