package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/oyetanishq/yappr/apps/agent/internal/handler"
	"github.com/oyetanishq/yappr/apps/agent/internal/middleware"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/db"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// ── Logger ──────────────────────────────────────────────────────────────
	log, _ := zap.NewProduction()
	defer log.Sync()

	// ── Config ──────────────────────────────────────────────────────────────
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	// ── Redis ───────────────────────────────────────────────────────────────
	rdb, err := db.NewRedis(cfg.Redis)
	if err != nil {
		log.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer rdb.Close()
	log.Info("redis connected", zap.String("addr", cfg.Redis.Addr))

	// ── Mongo ───────────────────────────────────────────────────────────────
	client, err := db.NewMongo(cfg.Mongo)
	if err != nil {
		log.Fatal("failed to connect to mongo", zap.Error(err))
	}
	defer client.Disconnect(context.Background())
	log.Info("mongo connected", zap.String("addr", cfg.Mongo.URI))

	// ── Router ──────────────────────────────────────────────────────────────
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.Logger(log))
	r.Use(middleware.Recovery(log))
	r.Use(middleware.CORS(cfg.App.AllowedOrigins))

	// ── Routes ──────────────────────────────────────────────────────────────
	handler.Register(r, rdb, client, log, cfg)

	// ── Server ──────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("server starting", zap.String("port", cfg.App.Port), zap.String("env", cfg.App.Env))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// ── Graceful Shutdown ────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("forced shutdown", zap.Error(err))
	}
	log.Info("server exited")
}
