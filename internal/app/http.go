package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type HTTPServer struct {
	srv *http.Server
}

func NewHTTPServer(handler http.Handler, port string) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr:    ":" + port,
			Handler: handler,
		},
	}
}

func (s *HTTPServer) Start() error {
	go func() {
		slog.Info("server_started", "addr", s.srv.Addr)

		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen_failed", "err", err)
			os.Exit(1)
		}
	}()

	return nil
}

func (s *HTTPServer) WaitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("server_stopping")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("server_stopped")
	return nil
}
