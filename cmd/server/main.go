package main

import (
	"avito-trainee-assignment/internal/server"
	"avito-trainee-assignment/internal/storage"
	"context"
	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"
	"log"
	"time"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap.NewDevelopment: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	sugar.Info("Application is starting")

	sugar.Info("Current time:", time.Now())

	cfg := server.EnvConfig{}
	if err := env.Parse(&cfg); err != nil {
		sugar.Fatalf("Cannot parse env config: %w", err)
	}

	store, err := storage.NewStore(context.Background(), sugar, storage.ConnectionTimeout(30*time.Second))
	if err != nil {
		sugar.Fatalf("Cannot create Store instance: %v", err)
	}

	serverOpts := []server.Option{
		server.WithEnvConfig(cfg),
		server.ReadTimeout(5 * time.Second),
	}

	srv, err := server.NewServer(sugar, store, serverOpts...)
	if err != nil {
		sugar.Fatalf("Cannot create Server instance: %v", err)
	}

	if err := srv.Start(); err != nil {
		sugar.Fatalf("Cannot start http srv: %v", err)
	}
}
