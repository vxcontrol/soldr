package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	_ "github.com/jinzhu/gorm/dialects/mysql" // GORM backend

	_ "soldr/internal/app/api/docs" // swagger docs
	"soldr/internal/app/api/utils"
	"soldr/internal/log"
)

type Config struct {
	Addr            string
	AddrHTTPS       string
	UseSSL          bool
	CertFile        string
	KeyFile         string
	GracefulTimeout time.Duration
}

type API struct {
	cfg    Config
	server *http.Server
	logger log.Logger
}

func NewAPI(cfg Config, router http.Handler, logger log.Logger) *API {
	api := &API{
		cfg:    cfg,
		server: &http.Server{Addr: cfg.Addr, Handler: router},
		logger: logger,
	}
	if cfg.UseSSL {
		api.server.Addr = cfg.AddrHTTPS
	}
	return api
}

func (s *API) Start() error {
	if s.cfg.UseSSL {
		go http.ListenAndServe(s.cfg.Addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, _ := net.SplitHostPort(r.Host)
			if host == "" {
				host = r.Host
			}
			if _, port, _ := net.SplitHostPort(utils.GetServerHost()); port != "" {
				r.Host = net.JoinHostPort(host, port)
			} else {
				r.Host = host
			}
			targetUrl := url.URL{Scheme: "https", Host: r.Host, Path: r.URL.Path, RawQuery: r.URL.RawQuery}
			http.Redirect(w, r, targetUrl.String(), http.StatusPermanentRedirect)
		}))

		if err := s.server.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %s", err)
		}
		return nil
	}

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %s", err)
	}
	return nil
}

func (s *API) Stop(err error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.GracefulTimeout)
	defer cancel()

	s.server.SetKeepAlivesEnabled(false)
	err = s.server.Shutdown(ctx)
	if err != nil {
		s.logger.WithError(err).Error("stop failure")
		return
	}
	s.logger.Info("gracefully stopped")
}
