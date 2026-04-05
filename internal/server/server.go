package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

func New(handler http.Handler, port string, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Start() error {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	serverErr := make(chan error, 1)

	go func() {
		s.logger.Info("server starting", "addr", s.httpServer.Addr)
		serverErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		s.logger.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			s.httpServer.Close()
			return fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	return nil
}
