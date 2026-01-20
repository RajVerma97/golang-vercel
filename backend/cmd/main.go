package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/RajVerma97/golang-vercel/backend/internal/app"
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

	// Start worker
	app.StartWorker(ctx)

	log.Println("Application running. Press Ctrl+C to stop.")
	// Wait for interrupt signal - THIS IS IMPORTANT!
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
