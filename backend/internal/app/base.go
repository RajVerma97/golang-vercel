package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/RajVerma97/golang-vercel/backend/internal/api/routes"
	"github.com/RajVerma97/golang-vercel/backend/internal/config"
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/RajVerma97/golang-vercel/backend/internal/server"
	"github.com/RajVerma97/golang-vercel/backend/internal/services"
)

type App struct {
	Config   *config.Config
	Server   *server.HTTPServer
	Services *services.Services
}

func NewApp() (*App, error) {
	config := config.NewConfig()
	ctx := context.Background()

	// services
	services, err := services.NewServices(ctx, config)
	if err != nil {
		logger.Error("failed to init services", err)
		return nil, err
	}

	// router
	router := routes.InitRouter(services)

	// server
	server, err := server.NewHTTPServer(router, config.Server)
	if err != nil {
		logger.Error("failed to init server", err)
		return nil, err
	}

	return &App{
		Config:   config,
		Server:   server,
		Services: services,
	}, nil
}

func (a *App) StartWorker(ctx context.Context) {
	go func() {
		logger.Debug("Starting worker")
		for {
			select {
			case <-ctx.Done():
				log.Println("Worker stopped due to context cancellation...")
				return
			default:
				build, err := a.Services.RedisService.DequeueBuild(ctx)
				if err != nil {
					log.Printf("Worker Error: %v", err)
					return
				}
				if build == nil {
					log.Println("no jobs in queue..Waiting")
					time.Sleep(2 * time.Second)
					continue
				}

				ctx := context.Background()
				a.ProcessJob(ctx, build)
			}
		}
	}()
}

func (a *App) ProcessJob(ctx context.Context, build *dto.Build) {
	cwd, _ := os.Getwd()
	tempDirPath := filepath.Join(cwd, "tmp", fmt.Sprintf("build-%d", build.ID))

	// Init environment
	if err := a.Services.WorkspaceManagerService.Create(ctx, build, tempDirPath); err != nil {
		logger.Error("failed to create workspace", err)
		return
	}

	defer func() {
		logger.Debug("Running defer ")
		if err := a.Services.WorkspaceManagerService.Cleanup(ctx, tempDirPath); err != nil {
			logger.Error("failed to cleanup workspace", err)
		}
	}()
	// Clone repo
	if err := a.Services.GitService.CloneRepository(ctx, build, tempDirPath); err != nil {
		logger.Error("failed to clone repo", err)
		return
	}

	// Build application
	if err := a.Services.BuildService.BuildApplication(ctx, build, tempDirPath); err != nil {
		logger.Error("failed to build", err)
		return
	}

	// Deploy application
	deployment := &dto.Deployment{
		BuildID:   build.ID,
		Status:    constants.DeploymentStatusPending,
		CreatedAt: time.Now(),
	}

	if err := a.Services.DeployService.DeployApplication(ctx, build, deployment, tempDirPath); err != nil {
		logger.Error("failed to deploy", err)
		return
	}
}
