package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/hector-leite/graph-policy-service/internal/config"
	"github.com/hector-leite/graph-policy-service/internal/server/handler"
	"github.com/hector-leite/graph-policy-service/internal/server/route"
)

type Server struct {
	logger           *slog.Logger
	cfg              *config.Config
	inferenceHandler *handler.InferenceHandler
	httpServer       *http.Server
}

type Option func(*Server)

func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) { s.logger = logger }
}

func WithConfig(cfg *config.Config) Option {
	return func(s *Server) { s.cfg = cfg }
}

func WithInferenceHandler(inferenceHandler *handler.InferenceHandler) Option {
	return func(s *Server) { s.inferenceHandler = inferenceHandler }
}

func NewServer(opts ...Option) *Server {
	s := &Server{}

	for _, opt := range opts {
		opt(s)
	}

	s.httpServer = &http.Server{
		Addr:    s.cfg.Server.Address,
		Handler: route.New(*s.inferenceHandler),
	}

	return s
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		s.logger.Info("Starting server", "address", s.cfg.Server.Address)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Could not start server", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	s.logger.Info("Shutting down server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Could not gracefully shutdown the server", "error", err)
		os.Exit(1)
	}

	s.logger.Info("Server stopped gracefully")
}
