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
	"github.com/docker/docker/api/types/container"
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
	// --------------------------------
	// BUILD PHASE
	// --------------------------------
	logger.Info("Starting Build Phase")
	dockerClient, err := docker_client.NewDockerClient()
	if err != nil {
		logger.Error("failed to create docker client", err)
		return
	}
	buildImageName := "golang:1.24-alpine"
	workDir := "/app"
	volumeBinds := []string{fmt.Sprintf("%s:/app", tempDirPath)}
	buildContainerName := fmt.Sprintf("build-worker-%d", job.ID)

	//When a build container with same buildContainerName already exists
	if dockerClient.DoesContainerExist(ctx, buildContainerName) {
		logger.Debug("Container name is taken, removing existing container...")
		// remove existing build container
		if err := dockerClient.RemoveContainer(ctx, buildContainerName); err != nil {
			logger.Error("failed to remove existing build container %s", err, zap.String("buildContainerName", buildContainerName))
			return
		}
	}

	// Create Build Container
	buildContainerId, err := dockerClient.CreateBuildContainer(ctx, buildImageName, buildContainerName, workDir, volumeBinds)
	if err != nil {
		logger.Error("failed to create build container", err)
		return
	}

	// Start Build Container
	err = dockerClient.StartContainer(ctx, buildContainerId)
	if err != nil {
		logger.Error("failed to start build container", err)
		return
	}

	// Wait for build to complete
	statusCh, errCh := dockerClient.WaitContainer(ctx, buildContainerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("Error waiting for container", err)
			return
		}
	case status := <-statusCh:
		logger.Info("Build container finished", zap.Int64("exit_code", status.StatusCode))

		if status.StatusCode != 0 {
			logs, _ := dockerClient.GetContainerLogs(ctx, buildContainerId)
			logger.Error("Build failed", nil, zap.String("logs", logs))
			return
		}
	}

	// Get build logs
	buildLogs, err := dockerClient.GetContainerLogs(ctx, buildContainerId)
	if err != nil {
		logger.Error("Failed to get container logs", err)
	} else {
		logger.Info("Build Output", zap.String("logs", buildLogs))
	}
	// Verify binary was created
	binaryPath := filepath.Join(tempDirPath, "bin", "app")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		logger.Error("Binary was not created", nil, zap.String("path", binaryPath))
		return
	}

	logger.Info("Build successful! Binary created", zap.String("path", binaryPath))
	// Clean up build container
	err = dockerClient.RemoveContainer(ctx, buildContainerId)
	if err != nil {
		logger.Warn("Failed to remove build container", zap.Error(err))
	}

	// --------------------------------
	// DEPLOYMENT PHASE
	// --------------------------------
	logger.Info("Starting Deployment Phase")

	//
	// Pull alpine for runtime
	deployImageName := "alpine:latest"
	err = dockerClient.PullImage(ctx, deployImageName)
	if err != nil {
		return
	}

	deployContainerName := fmt.Sprintf("deployment-%d", job.ID)

	// Check and remove existing deployment container (ADD THIS)
	if dockerClient.DoesContainerExist(ctx, deployContainerName) {
		logger.Debug("Deployment container already exists, removing...")
		if err := dockerClient.RemoveContainer(ctx, deployContainerName); err != nil {
			logger.Error("failed to remove existing deployment container", err,
				zap.String("deployContainerName", deployContainerName))
			return
		}
	}
	deployVolumeBinds := []string{fmt.Sprintf("%s/bin:/app", tempDirPath)}

	deployContainerID, err := dockerClient.CreateDeploymentContainer(
		ctx,
		deployImageName,
		deployContainerName,
		deployVolumeBinds,
		"8080", // Your app's port
		int(job.ID),
	)
	if err != nil {
		logger.Error("failed to create deployment container", err)
		return
	}

	err = dockerClient.StartContainer(ctx, deployContainerID)
	if err != nil {
		logger.Error("failed to start deployment container", err)
		return
	}

	// After starting the deployment container, get its logs
	time.Sleep(2 * time.Second) // Give it a moment to start

	deployLogs, err := dockerClient.GetContainerLogs(ctx, deployContainerID)
	if err != nil {
		logger.Error("Failed to get deployment logs", err)
	} else {
		logger.Info("Deployment Container Logs", zap.String("logs", deployLogs))
	}

	// Also check container status
	inspect, err := dockerClient.InspectContainer(ctx, deployContainerID)
	if err != nil {
		logger.Error("Failed to inspect deployment container", err)
		return
	}

	logger.Info("Container State",
		zap.Bool("running", inspect.State.Running),
		zap.String("status", inspect.State.Status),
		zap.Int("exit_code", inspect.State.ExitCode),
		zap.String("error", inspect.State.Error))

	// Get the assigned port
	inspect, err = dockerClient.InspectContainer(ctx, deployContainerID)
	if err != nil {
		logger.Error("Failed to inspect deployment container", err)
		return
	}

	// Check if container is still running
	if !inspect.State.Running {
		logger.Error("Deployment container exited unexpectedly",
			nil,
			zap.Int("exit_code", inspect.State.ExitCode),
			zap.String("error", inspect.State.Error))

		// Get logs to see why it exited
		logs, _ := dockerClient.GetContainerLogs(ctx, deployContainerID)
		logger.Error("Container logs", nil, zap.String("logs", logs))
		return
	}

	// Check if port bindings exist
	portBindings, exists := inspect.NetworkSettings.Ports["8080/tcp"]
	if !exists || len(portBindings) == 0 {
		logger.Error("No port bindings found for container", nil)
		return
	}

	hostPort := portBindings[0].HostPort
	deploymentURL := fmt.Sprintf("http://localhost:%s", hostPort)

	logger.Info("🚀 Deployment successful!",
		zap.String("url", deploymentURL),
		zap.String("containerID", deployContainerID))

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
