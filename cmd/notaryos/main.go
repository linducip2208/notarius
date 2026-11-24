// Command notaryos starts the NOTARYOS modular monolith.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"notaryos/internal/app"
	"notaryos/internal/config"
	"notaryos/internal/database"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg := config.Load()

	if !cfg.TLSCookie {
		log.Warn("COOKIE_SECURE=false — wajib true di produksi (TLS/nginx + certbot). Lihat DEPLOYMENT_GUIDE langkah 6.")
	}
	if cfg.AppBaseURL == "" {
		log.Warn("APP_BASE_URL kosong — QR kuitansi hanya berisi kode, isi dengan https://domain-notaris di produksi.")
	}

	db, err := database.Open(cfg.DSN)
	if err != nil {
		log.Error("database unavailable — refusing to start with fake data", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	applied, err := database.Migrate(db, os.DirFS("migrations"), ".")
	if err != nil {
		log.Error("migration failed", "err", err)
		os.Exit(1)
	}
	log.Info("migrations applied", "count", applied)

	a, err := app.New(cfg, db, log)
	if err != nil {
		log.Error("app init failed", "err", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           a.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Info("notaryos listening", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server stopped", "err", err)
			os.Exit(1)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown initiated")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "err", err)
		os.Exit(1)
	}
	log.Info("server stopped cleanly")
}
