package server

import (
	"avito-trainee-assignment/internal/storage"
	"context"
	"errors"
	"fmt"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
)

// Server defines fields used in HTTP processing.
type Server struct {
	logger        *zap.SugaredLogger
	httpServer    *http.Server
	afterShutdown []func()
}

// NewServer constructs a Server. See the various Options for available customizations.
func NewServer(logger *zap.SugaredLogger, store *storage.Store, opts ...Option) (*Server, error) {
	if logger == nil {
		return nil, errors.New("no logger provided")
	}

	if store == nil {
		return nil, errors.New("no store provided")
	}

	cfg := &config{httpServer: &http.Server{}}

	// setting application-specific default handlers
	h := handler{
		logger: logger,
		store:  store,
		parsers: parsers{
			createChatPool:       fastjson.ParserPool{},
			createMessagePool:    fastjson.ParserPool{},
			chatsByUserIDPool:    fastjson.ParserPool{},
			messagesByChatIDPool: fastjson.ParserPool{},
		},
	}

	defaultHandlers := map[string]http.Handler{
		"/users/add":    http.HandlerFunc(h.createUser),
		"/chats/add":    http.HandlerFunc(h.createChat),
		"/messages/add": http.HandlerFunc(h.createMessage),
		"/chats/get":    http.HandlerFunc(h.chatsByUserID),
		"/messages/get": http.HandlerFunc(h.messagesByChatID),
	}

	cfg.handlers = defaultHandlers

	// extending given options with mandatory
	opts = append(
		opts,
		applyEnforcePostJson(),
		applyLog(logger.Desugar()),
		registerHandlers(),
		RegisterAfterShutdown(func() {
			store.Close()
		}),
	)

	// applying options
	for _, o := range opts {
		o.apply(cfg)
	}

	srv := &Server{
		logger:        logger,
		httpServer:    cfg.httpServer,
		afterShutdown: cfg.afterShutdown,
	}

	return srv, nil
}

// Start calls ListenAndServe on http.Server instance inside Server struct
// and implements graceful shutdown via goroutine waiting for signals.
func (s *Server) Start() error {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		s.logger.Info("Shutting down HTTP server")

		if err := s.httpServer.Shutdown(context.Background()); err != nil {
			s.logger.Errorf("srv.Shutdown: %v", err)
		}
		s.logger.Info("HTTP server is stopped")

		close(idleConnsClosed)
	}()

	s.logger.Infof("Starting HTTP server on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("s.httpServer.ListenAndServe: %v", err)
	}

	<-idleConnsClosed

	for _, f := range s.afterShutdown {
		f()
	}

	return nil
}
