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
	"github.com/RajVerma97/golang-vercel/backend/internal/constants"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/RajVerma97/golang-vercel/backend/internal/server"
	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

type App struct {
	Config       *config.Config
	Server       *server.HTTPServer
	RedisClient  *redis_client.RedisClient
	DockerClient *docker_client.DockerClient
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

	// docker client
	dockerClient, err := docker_client.NewDockerClient()
	if err != nil {
		return nil, err
	}

	return &App{
		Config:       config,
		Server:       server,
		RedisClient:  nil,
		DockerClient: dockerClient,
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

func (a *App) InitializeEnvironment(ctx context.Context, build *dto.Build, tempDirPath string) error {
	// remove existing /tmp/build-%d directory
	if _, err := os.Stat(tempDirPath); err == nil {
		logger.Debug("Removing existing temp directory", zap.String("path", tempDirPath))
		err = os.RemoveAll(tempDirPath)
		if err != nil {
			logger.Error("Failed to remove existing temp directory", err)
			return fmt.Errorf("failed to remove existing temp directory;%w", err)
		}
	}
	// create fresh directory
	err := os.MkdirAll(tempDirPath, 0755)
	if err != nil {
		logger.Error("Failed to create directory", err, zap.String("tempDirPath", tempDirPath))
		return fmt.Errorf("failed to create directory:%w", err)
	}
	logger.Debug("Successfully created temp dir", zap.String("tempDirPath", tempDirPath))
	return nil
}

func (a *App) CloneRepository(ctx context.Context, build *dto.Build, tempDirPath string) error {
	// Execute: git clone <repo_url> <temp_dir>
	args := []string{"clone"}

	// If branch is provided, add "-b branchName"
	if build.Branch != "" {
		args = append(args, "-b", build.Branch)
	}

	// Add repo URL and destination
	args = append(args, build.RepoUrl, tempDirPath)

	cloneCmd := exec.CommandContext(ctx, "git", args...)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		logger.Error("Git clone failed", err)
		return fmt.Errorf("failed to git clone:%w", err)
	}
	// logger.Debug("Successfully cloned Repository")

	// 2. CHECKOUT COMMIT HASH(if provided)
	if build.CommitHash != "" {
		checkoutCmd := exec.CommandContext(ctx, "git", "checkout", build.CommitHash)
		checkoutCmd.Dir = tempDirPath
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr

		if err := checkoutCmd.Run(); err != nil {
			logger.Error("Git checkout failed", err)
			return fmt.Errorf("failed to checkout commit %s: %w", build.CommitHash, err)
		}
	}
	logger.Debug("Successfully cloned Repository", zap.String("branch", build.Branch), zap.String("hash", build.CommitHash))
	return nil
}

func (a *App) BuildApplication(ctx context.Context, build *dto.Build, tempDirPath string) error {
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

func (a *App) DeployApplication(ctx context.Context, build *dto.Build, deployment *dto.Deployment, tempDirPath string) error {
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

	logger.Info("🚀 Deployment successful!",
		zap.String("url", deploymentURL),
		zap.String("containerID", deployContainerID))

	deployment.URL = deploymentURL
	deployment.Status  = constants.DeploymentStatusRunning
	return nil
}

func (a *App) ProcessJob(ctx context.Context, build *dto.Build) error {
	logger.Debug("Processing Job", zap.Any("job", build))
	// mark job as building
	build.Status = constants.BuildStatusBuilding
	cwd, _ := os.Getwd()
	tempDirPath := filepath.Join(cwd, "tmp", fmt.Sprintf("build-%d", build.ID))

	// 1. Initialize ENVIRONMENT
	if err := a.InitializeEnvironment(ctx, build, tempDirPath); err != nil {
		// mark the job as failed
		build.Status = constants.BuildStatusFailed
		return fmt.Errorf("environment initialization failed: %w", err)
	}

	// 2. Clone Repository
	if err := a.CloneRepository(ctx, build, tempDirPath); err != nil {
		// mark the job as failed
		build.Status = constants.BuildStatusFailed
		return fmt.Errorf("repository clone failed: %w", err)
	}

	// 3. Build App

	if err := a.BuildApplication(ctx, build, tempDirPath); err != nil {
		// mark. the job as failed
		build.Status = constants.BuildStatusFailed
		return fmt.Errorf("application build failed: %w", err)
	}

	logger.Debug("after building", zap.Any("build", build))

	deployment := &dto.Deployment{
		BuildID:   build.ID,
		Status:    constants.DeploymentStatusPending,
		CreatedAt: time.Now(),
	}

	// 4. Deploy App
	if err := a.DeployApplication(ctx, build, deployment, tempDirPath); err != nil {
		// mark the build as failed
		build.Status = constants.BuildStatusFailed
		return fmt.Errorf("application deployment failed: %w", err)
	}
	logger.Debug("after deploying", zap.Any("deployment", deployment))

	// mark the build as  success
	build.Status = constants.BuildStatusSuccess
	return nil
}
