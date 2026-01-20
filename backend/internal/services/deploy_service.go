package services

import (
	"context"
	"fmt"
	"time"

	docker_client "github.com/RajVerma97/golang-vercel/backend/internal/client/docker"
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"go.uber.org/zap"
)

type DeployServiceConfig struct {
	DockerClient *docker_client.DockerClient
}
type DeployService struct {
	DockerClient *docker_client.DockerClient
}

func NewDeployService(config *DeployServiceConfig) *DeployService {
	return &DeployService{
		DockerClient: config.DockerClient,
	}
}

func (a *DeployService) DeployApplication(ctx context.Context, build *dto.Build, deployment *dto.Deployment, tempDirPath string) error {
	logger.Info("Starting Deployment Phase")

	// Pull alpine for runtime
	deployImageName := "alpine:latest"
	err := a.DockerClient.PullImage(ctx, deployImageName)
	if err != nil {
		return fmt.Errorf("failed to pull deploy image: %w", err)
	}

	deployContainerName := fmt.Sprintf("deployment-%d", build.ID)

	// Check and remove existing deployment container (ADD THIS)
	if a.DockerClient.DoesContainerExist(ctx, deployContainerName) {
		logger.Debug("Deployment container already exists, removing...")
		if err := a.DockerClient.RemoveContainer(ctx, deployContainerName); err != nil {
			logger.Error("failed to remove existing deployment container", err,
				zap.String("deployContainerName", deployContainerName))
			return err
		}
	}
	deployVolumeBinds := []string{fmt.Sprintf("%s/bin:/app", tempDirPath)}

	deployContainerID, err := a.DockerClient.CreateDeploymentContainer(
		ctx,
		deployImageName,
		deployContainerName,
		deployVolumeBinds,
		"8080", // Your app's port
		int(build.ID),
	)
	if err != nil {
		logger.Error("failed to create deployment container", err)
		return err
	}

	err = a.DockerClient.StartContainer(ctx, deployContainerID)
	if err != nil {
		logger.Error("failed to start deployment container", err)
		return err
	}
	if deployment.Container == nil {
		deployment.Container = &dto.Container{}
	}
	deployment.Container.ID = deployContainerID
	deployment.Container.Name = deployContainerName

	// After starting the deployment container, get its logs
	time.Sleep(2 * time.Second) // Give it a moment to start

	deployLogs, err := a.DockerClient.GetContainerLogs(ctx, deployContainerID)
	if err != nil {
		logger.Error("Failed to get deployment logs", err)
	} else {
		logger.Info("Deployment Container Logs", zap.String("logs", deployLogs))
	}
	deployment.Logs = deployLogs

	// Also check container status
	inspect, err := a.DockerClient.InspectContainer(ctx, deployContainerID)
	if err != nil {
		logger.Error("Failed to inspect deployment container", err)
		return fmt.Errorf("failed to inspect deployment container: %w", err)
	}

	logger.Info("Container State",
		zap.Bool("running", inspect.State.Running),
		zap.String("status", inspect.State.Status),
		zap.Int("exit_code", inspect.State.ExitCode),
		zap.String("error", inspect.State.Error))

	// Check if container is still running
	if !inspect.State.Running {
		logger.Error("Deployment container exited unexpectedly",
			nil,
			zap.Int("exit_code", inspect.State.ExitCode),
			zap.String("error", inspect.State.Error))

		// Get logs to see why it exited
		logs, _ := a.DockerClient.GetContainerLogs(ctx, deployContainerID)
		logger.Error("Container logs", nil, zap.String("logs", logs))
		return fmt.Errorf("deployment container exited unexpectedly (exit code: %d): %s",
			inspect.State.ExitCode, inspect.State.Error)
	}

	// Check if port bindings exist
	portBindings, exists := inspect.NetworkSettings.Ports["8080/tcp"]
	if !exists || len(portBindings) == 0 {
		logger.Error("No port bindings found for container", nil)
		return fmt.Errorf("no port bindings found for container")
	}

	hostPort := portBindings[0].HostPort
	deploymentURL := fmt.Sprintf("http://localhost:%s", hostPort)

	logger.Info("✅ Deployment successful!",
		zap.String("url", deploymentURL),
		zap.String("containerID", deployContainerID))

	deployment.URL = deploymentURL
	deployment.Status = constants.DeploymentStatusRunning
	return nil
}
