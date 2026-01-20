package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/app"
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
)

func main() {
	app, err := app.NewApp()
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	// Start server
	err = app.Server.Start(ctx)
	if err != nil {
		panic(err)
	}

	// start worker
	app.StartWorker(ctx)

	//  enqueue
	err = app.Services.RedisService.EnqueueBuild(ctx, &dto.Build{
		ID:         1,
		RepoUrl:    "https://github.com/RajVerma97/golang-demo.git",
		CommitHash: "52690c286cf237ef76520b088b787354dd2df80e",
		Branch:     "master",
		Status:     constants.BuildStatusPending,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		panic(err)
	}

	log.Println("Application running. Press Ctrl+C to stop.")
	// Wait for interrupt signal - THIS IS IMPORTANT!
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
