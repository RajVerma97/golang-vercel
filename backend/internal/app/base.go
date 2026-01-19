package app

import (
	"context"
	"log"
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/client"
	"github.com/RajVerma97/golang-vercel/backend/internal/config"
	"github.com/RajVerma97/golang-vercel/backend/internal/server"
)

type App struct {
	Config      *config.Config
	Server      *server.HTTPServer
	RedisClient *client.RedisClient
}

func NewApp() (*App, error) {
	config := config.NewConfig()
	server, err := server.NewHTTPServer(config.Server)
	if err != nil {
		return nil, err
	}
	redisClient, err := client.NewRedisClient(config.Redis)
	if err != nil {
		return nil, err
	}
	return &App{
		Config:      config,
		Server:      server,
		RedisClient: redisClient,
	}, nil
}

func (a *App) StartWorker(ctx context.Context) {
	go func() {
		log.Println("Starting worker")
		for {
			select {
			case <-ctx.Done():
				log.Println("Worker stopped due to context cancellation...")
				return
			default:
				job, err := a.RedisClient.DequeueBuild(ctx)
				if err != nil {
					log.Printf("Worker Error: %v", err)
					return
				}
				if job == nil {
					log.Println("no jobs in queue..Waiting")
					time.Sleep(2 * time.Second)
					continue
				}
				log.Printf("PROCESSING JOB:%d", job.ID)
			}
		}
	}()

}
