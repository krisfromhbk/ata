package server

import (
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"time"
)

type Option interface {
	apply(*config)
}

type optionFunc func(c *config)

func (f optionFunc) apply(c *config) { f(c) }

// config defines fields used for configuring Server instance
type config struct {
	httpServer    *http.Server
	handlers      map[string]http.Handler
	afterShutdown []func()
}

// EnvConfig defines fields used for parsing from environment variables
type EnvConfig struct {
	Host string `env:"HOST" envDefault:"0.0.0.0"`
	Port uint16 `env:"PORT" envDefault:"9000"`
}

// WithEnvConfig enables processing exported EnvConfig struct to acts as a source of config parameters for http.Server
func WithEnvConfig(cfg EnvConfig) Option {
	return optionFunc(func(c *config) {
		c.httpServer.Addr = cfg.Host + ":" + strconv.FormatUint(uint64(cfg.Port), 10)
	})
}

// ReadTimeout sets read timeout for http.Server
func ReadTimeout(d time.Duration) Option {
	return optionFunc(func(c *config) {
		c.httpServer.ReadTimeout = d
	})
}

// RegisterAfterShutdown registers a function to call after http.Server shutdown
// f will not be called in separated goroutine
func RegisterAfterShutdown(f func()) Option {
	return optionFunc(func(c *config) {
		c.afterShutdown = append(c.afterShutdown, f)
	})
}

// registerHandlers iterates over a handlers map and registers each handler for newly initialized http.ServeMux
// that http.ServeMux is used as a http.Handler for http.Server in config struct
func registerHandlers() Option {
	return optionFunc(func(c *config) {
		mux := http.NewServeMux()
		for pattern, h := range c.handlers {
			mux.Handle(pattern, h)
		}
		c.httpServer.Handler = mux
	})
}

// applyEnforcePostJson wraps each handler in handlers map with enforcePostJson middleware
func applyEnforcePostJson() Option {
	return optionFunc(func(c *config) {
		for pattern, h := range c.handlers {
			c.handlers[pattern] = enforcePostJson(h)
		}
	})
}

// applyLog wraps each http.Handler in handlers map with log middleware
func applyLog(logger *zap.Logger) Option {
	return optionFunc(func(c *config) {
		for pattern, h := range c.handlers {
			c.handlers[pattern] = log(h, logger)
		}
	})
}

// TimeoutHandler wraps each handler in handlers map in http.TimeoutHandler with provided duration and message
func TimeoutHandler(d time.Duration, msg string) Option {
	return optionFunc(func(c *config) {
		for pattern, h := range c.handlers {
			c.handlers[pattern] = http.TimeoutHandler(h, d, msg)
		}
	})
}
