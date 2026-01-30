package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// ================================
// Config
// ================================

type Config struct {
	Address      string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// TLS
	EnableTLS bool
	CertFile  string
	KeyFile   string
}

// ================================
// Server
// ================================

type Server struct {
	httpServer *http.Server
	cfg        Config
}

func New(cfg Config, handler http.Handler) *Server {
	srv := &http.Server{
		Addr:         cfg.Address + ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return &Server{
		httpServer: srv,
		cfg:        cfg,
	}
}

// ================================
// Start
// ================================

func (s *Server) Start() error {
	log.Printf("HTTP server starting on %s\n", s.httpServer.Addr)

	if s.cfg.EnableTLS {
		if s.cfg.CertFile == "" || s.cfg.KeyFile == "" {
			return errors.New("tls enabled but cert/key file missing")
		}
		return s.httpServer.ListenAndServeTLS(s.cfg.CertFile, s.cfg.KeyFile)
	}

	return s.httpServer.ListenAndServe()
}

// ================================
// Graceful Shutdown
// ================================

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("HTTP server shutting down...")
	return s.httpServer.Shutdown(ctx)
}

// ================================
// Production Runner
// ================================

func Run(cfg Config, handler http.Handler) error {
	srv := New(cfg, handler)

	// OS Signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("server exited cleanly")
	return nil
}
