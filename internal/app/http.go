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
	srv    *http.Server
	notify chan error
}

func NewHTTPServer(handler http.Handler, port string) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr:         ":" + port,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		notify: make(chan error, 1),
	}
}

func (s *HTTPServer) Start() error {
	go func() {
		slog.Info("server_started", "addr", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil {
			s.notify <- err
		}
		close(s.notify)
	}()
	return nil
}

func (s *HTTPServer) Notify() <-chan error {
	return s.notify
}

func (s *HTTPServer) WaitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("server_stopping")
	case err := <-s.notify:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.srv.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("server_stopped")
	return nil
}
