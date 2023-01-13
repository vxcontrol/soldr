package server

import (
	"context"
	"net/http"
	"time"

	_ "github.com/jinzhu/gorm/dialects/mysql" // GORM backend

	_ "soldr/pkg/app/api/docs" // swagger docs
)

// Server is a wrapper of standard http.Server with cancellation (via context)
// and graceful shutdown logic.
type Server struct {
	Addr            string
	CertFile        string
	KeyFile         string
	GracefulTimeout time.Duration
}

func (s Server) ListenAndServe(ctx context.Context, router http.Handler) error {
	srv := &http.Server{Addr: s.Addr, Handler: router}
	return s.run(ctx, srv, srv.ListenAndServe)
}

func (s Server) ListenAndServeTLS(ctx context.Context, router http.Handler) error {
	srv := &http.Server{Addr: s.Addr, Handler: router}
	return s.run(ctx, srv, func() error {
		return srv.ListenAndServeTLS(s.CertFile, s.KeyFile)
	})
}

func (s Server) run(ctx context.Context, srv *http.Server, serve func() error) error {
	done := make(chan error, 1)
	go func() { done <- serve() }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		ctx, cancel := context.WithTimeout(context.Background(), s.GracefulTimeout)
		defer cancel()
		srv.SetKeepAlivesEnabled(false)
		return srv.Shutdown(ctx)
	}
}
