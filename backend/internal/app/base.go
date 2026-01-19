package app

import (
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
