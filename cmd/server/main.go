package main

import (
	"avito-trainee-assignment/internal/server"
	"avito-trainee-assignment/internal/storage"
	"go.uber.org/zap"
	"log"
)

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("zap.NewDevelopment: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	sugar.Info("Application is starting")

	store, err := storage.NewStore(sugar, storage.TestConfig)
	if err != nil {
		sugar.Fatalf("Cannot create Store instance: %v", err)
	}

	srv, err := server.NewServer(sugar, server.Config{Port: 9000, Host: ""}, store)
	if err != nil {
		sugar.Fatalf("Cannot create Server instance: %v", err)
	}

	if err := srv.Start(); err != nil {
		sugar.Fatalf("Cannot start http srv: %v", err)
	}
}
