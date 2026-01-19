package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	docker_client "github.com/RajVerma97/golang-vercel/backend/internal/client/docker"
	redis_client "github.com/RajVerma97/golang-vercel/backend/internal/client/redis"
	"github.com/RajVerma97/golang-vercel/backend/internal/config"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/RajVerma97/golang-vercel/backend/internal/server"
	"go.uber.org/zap"
)

type App struct {
	Config      *config.Config
	Server      *server.HTTPServer
	RedisClient *redis_client.RedisClient
}

func NewApp() (*App, error) {
	config := config.NewConfig()

	// logger
	err := logger.Init("development")

	// server
	server, err := server.NewHTTPServer(config.Server)
	if err != nil {
		return nil, err
	}

	// redis
	// redisClient, err := redis_client.NewRedisClient(config.Redis)
	// if err != nil {
	// 	return nil, err
	// }

	return &App{
		Config:      config,
		Server:      server,
		RedisClient: nil,
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

				a.ProcessJob(ctx, job)
			}
		}
	}()

}

func CreateDirectory(path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return nil
}
func (a *App) ProcessJob(ctx context.Context, job *dto.Job) {
	logger.Debug("Processing Job", zap.Any("job", job))
	// 1. INITIALIZE ENVIRONMENT

	// delete existing temp dir
	cwd, _ := os.Getwd()
	tempDirPath := filepath.Join(cwd, "tmp", fmt.Sprintf("build-%d", job.ID))

	// remove existing /tmp/build-%d directory
	if _, err := os.Stat(tempDirPath); err == nil {
		logger.Debug("Removing existing temp directory", zap.String("path", tempDirPath))
		err = os.RemoveAll(tempDirPath)
		if err != nil {
			logger.Error("Failed to remove existing temp directory", err)
			return
		}
	}
	// create fresh directory
	err := os.MkdirAll(tempDirPath, 0755)
	if err != nil {
		logger.Error("Failed to create directory", err, zap.String("tempDirPath", tempDirPath))
		return
	}
	logger.Debug("Successfully created temp dir", zap.String("tempDirPath", tempDirPath))

	// 2. CLONE REPOSITORY
	// Execute: git clone <repo_url> <temp_dir>
	cloneCmd := exec.CommandContext(ctx, "git", "clone", job.RepoUrl, tempDirPath)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	err = cloneCmd.Run()
	if err != nil {
		logger.Error("Git clone failed", err)
		return
	}
	logger.Debug("Successfully cloned Repository")

	// - Ensure you handle private repos if necessary via SSH keys or Tokens.

	// 3. START ISOLATED CONTAINER (The "Build" Box)
	// - Spin up a Docker container (e.g., using the Docker Go SDK).
	dockerClient, err := docker_client.NewDockerClient()
	if err != nil {
		logger.Error("failed to create docker client", err)
		return
	}

	_, err = dockerClient.CreateNewContainer("redis")
	if err != nil {
		logger.Error("failed to create pull image", err)
		return
	}

	// - Mount the <temp_dir> as a volume inside the container.
	// - Use a base image like 'node:alpine' or 'python:slim' depending on the framework.

	// 4. INSTALL & BUILD
	// - Inside the container, run: 'npm install && npm run build'
	// - Capturing the output (stdout/stderr) is critical for debugging.
	// - Identify the output folder (usually 'dist', 'build', or '.next').

	// 5. UPLOAD ARTIFACTS
	// - Push the generated static files to a Storage Provider (AWS S3, Google Cloud Storage).
	// - The folder structure should follow the deployment ID: /deployments/<job-id>/*

	// 6. UPDATE DATABASE & STATUS
	// - Update MongoDB: status = 'COMPLETED', deployment_url = 'https://<job-id>.yourdomain.com'
	// - Clear the local temporary directory to save disk space.
}
