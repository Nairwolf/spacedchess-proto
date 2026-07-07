package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nairwolf/spacedchess/internal/migrate"
	"github.com/nairwolf/spacedchess/internal/server"
	"github.com/nairwolf/spacedchess/internal/store"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}
	addr := env("ADDR", ":8080")
	staticDir := os.Getenv("STATIC_DIR")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// The database container may still be starting; retry briefly.
	var pool *pgxpool.Pool
	var err error
	for {
		pool, err = pgxpool.New(ctx, databaseURL)
		if err == nil {
			err = pool.Ping(ctx)
		}
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			log.Error("could not connect to database", "err", err)
			os.Exit(1)
		case <-time.After(time.Second):
			log.Info("waiting for database", "err", err)
		}
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool); err != nil {
		log.Error("migrations failed", "err", err)
		os.Exit(1)
	}

	srv := server.New(store.New(pool), server.Options{
		StaticDir:         staticDir,
		SecureCookies:     env("SECURE_COOKIES", "false") == "true",
		AllowRegistration: env("ALLOW_REGISTRATION", "true") != "false",
	}, log)

	log.Info("listening", "addr", addr, "static_dir", staticDir)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Error("server exited", "err", err)
		os.Exit(1)
	}
}
