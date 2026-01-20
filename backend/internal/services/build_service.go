package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	docker_client "github.com/RajVerma97/golang-vercel/backend/internal/client/docker"
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

type BuildServiceConfig struct {
	DockerClient *docker_client.DockerClient
}
type BuildService struct {
	DockerClient *docker_client.DockerClient
}

func NewBuildService(config *BuildServiceConfig) *BuildService {
	return &BuildService{
		DockerClient: config.DockerClient,
	}
}

func (a *BuildService) BuildApplication(ctx context.Context, build *dto.Build, tempDirPath string) error {
	// mark the build as building
	now := time.Now()
	build.StartedAt = &now
	build.Status = constants.BuildStatusBuilding
	logger.Info("Starting Build Phase")
	buildImageName := "golang:1.24-alpine"
	workDir := "/app"
	volumeBinds := []string{fmt.Sprintf("%s:/app", tempDirPath)}
	buildContainerName := fmt.Sprintf("build-worker-%d", build.ID)

	//When a build container with same buildContainerName already exists
	if a.DockerClient.DoesContainerExist(ctx, buildContainerName) {
		logger.Debug("Container name is taken, removing existing container...")
		// remove existing build container
		if err := a.DockerClient.RemoveContainer(ctx, buildContainerName); err != nil {
			logger.Error("failed to remove existing build container %s", err, zap.String("buildContainerName", buildContainerName))
			return fmt.Errorf("failed to remove existing build container:%w", err)
		}
	}

	// Create Build Container
	buildContainerId, err := a.DockerClient.CreateBuildContainer(ctx, buildImageName, buildContainerName, workDir, volumeBinds)
	if err != nil {
		logger.Error("failed to create build container", err)
		return fmt.Errorf("failed to create build container:%w", err)
	}

	// Start Build Container
	err = a.DockerClient.StartContainer(ctx, buildContainerId)
	if err != nil {
		logger.Error("failed to start build container", err)
		return fmt.Errorf("failed to start build container:%w", err)
	}
	if build.Container == nil {
		build.Container = &dto.Container{}
	}
	build.Container.ID = buildContainerId
	build.Container.Name = buildContainerName

	// Wait for build to complete
	statusCh, errCh := a.DockerClient.WaitContainer(ctx, buildContainerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			logger.Error("Error waiting for container", err)
			return fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		logger.Info("Build container finished", zap.Int64("exit_code", status.StatusCode))

		if status.StatusCode != 0 {
			logs, _ := a.DockerClient.GetContainerLogs(ctx, buildContainerId)
			logger.Error("Build failed", nil, zap.String("logs", logs))
			return fmt.Errorf("build failed with exit code %d: %s", status.StatusCode, logs)
		}
	}

	// Get build logs
	buildLogs, err := a.DockerClient.GetContainerLogs(ctx, buildContainerId)
	if err != nil {
		logger.Error("Failed to get container logs", err)
	} else {
		logger.Info("Build Output", zap.String("logs", buildLogs))
	}
	build.Logs = buildLogs
	// Verify binary was created
	binaryPath := filepath.Join(tempDirPath, "bin", "app")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		logger.Error("Binary was not created", nil, zap.String("path", binaryPath))
		return fmt.Errorf("binary was not created at %s", binaryPath)
	}
	build.BinaryPath = &binaryPath

	logger.Info("Build successful! Binary created", zap.String("path", binaryPath))
	// Clean up build container
	err = a.DockerClient.RemoveContainer(ctx, buildContainerId)
	if err != nil {
		logger.Warn("Failed to remove build container", zap.Error(err))
	}
	build.Status = constants.BuildStatusSuccess
	build.CompletedAt = &now
	return nil
}
