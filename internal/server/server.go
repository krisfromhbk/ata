package server

import (
	"avito-trainee-assignment/internal/storage"
	"context"
	"fmt"
	"github.com/valyala/fastjson"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"strconv"
)

// Server defines fields used in HTTP processing
type Server struct {
	logger     *zap.SugaredLogger
	httpServer *http.Server
	h          handler
}

// NewServer returns new Server struct with provided zap.SugaredLogger and storage.Store
func NewServer(logger *zap.SugaredLogger, config Config, store *storage.Store) (*Server, error) {
	srv := &Server{
		logger:     logger,
		httpServer: nil,
		h: handler{
			logger: logger,
			store:  store,
			parsers: parsers{
				createChatPool:       fastjson.ParserPool{},
				createMessagePool:    fastjson.ParserPool{},
				chatsByUserIDPool:    fastjson.ParserPool{},
				messagesByChatIDPool: fastjson.ParserPool{},
			},
		},
	}

	mux := http.NewServeMux()
	mux.Handle("/users/add", enforcePOSTJSON(http.HandlerFunc(srv.h.createUser)))
	mux.Handle("/chats/add", enforcePOSTJSON(http.HandlerFunc(srv.h.createChat)))
	mux.Handle("/messages/add", enforcePOSTJSON(http.HandlerFunc(srv.h.createMessage)))
	mux.Handle("/chats/get", enforcePOSTJSON(http.HandlerFunc(srv.h.chatsByUserID)))
	mux.Handle("/messages/get", enforcePOSTJSON(http.HandlerFunc(srv.h.messagesByChatID)))

	httpServer := &http.Server{
		Addr:    config.Host + ":" + strconv.FormatUint(uint64(config.Port), 10),
		Handler: mux,
	}

	srv.httpServer = httpServer

	return srv, nil
}

// Start calls ListenAndServe on http.Server instance inside Server struct
// and implements graceful shutdown via goroutine waiting for signals
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

	s.logger.Info("Closing store")
	s.h.store.Close()
	s.logger.Info("Store is closed")

	return nil
}
