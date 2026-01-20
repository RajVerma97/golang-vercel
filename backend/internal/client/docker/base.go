package docker_client

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

type DockerClient struct {
	client *client.Client
}

func NewDockerClient() (*DockerClient, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		client: dockerClient,
	}, nil
}

func (c *DockerClient) Close() error {
	return c.client.Close()
}

func (c *DockerClient) PullImage(ctx context.Context, imageName string) error {
	logger.Debug("Pulling image", zap.String("image", imageName))

	reader, err := c.client.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		logger.Error("Failed to pull image", err)
		return err
	}
	defer reader.Close()

	// Wait for pull to complete
	io.Copy(os.Stdout, reader)

	logger.Debug("Successfully pulled image", zap.String("image", imageName))
	return nil
}

// 2. CREATE CONTAINER (with volume mounts for your build files)
func (c *DockerClient) CreateBuildContainer(ctx context.Context, imageName, containerName, workDir string, volumeBinds []string) (string, error) {
	logger.Debug("Creating container", zap.String("image", imageName))
	cmd := `
		set -e
		go mod tidy
		 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/app .
		`

	resp, err := c.client.ContainerCreate(ctx,
		&container.Config{
			Image:      imageName,
			WorkingDir: workDir,
			Cmd:        []string{"sh", "-c", cmd},
		},
		&container.HostConfig{
			Binds: volumeBinds, // e.g., ["/tmp/build-1:/app"]
		},
		nil, nil, containerName)

	if err != nil {
		logger.Error("Failed to create container", err)
		return "", err
	}

	logger.Debug("✅ Successfully created container", zap.String("container_id", resp.ID))
	return resp.ID, nil
}
func (c *DockerClient) CreateDeploymentContainer(ctx context.Context, imageName string, containerName string, volumeBinds []string, port string, deploymentID int) (string, error) {
	logger.Debug("Creating deployment container", zap.String("name", containerName))
	resp, err := c.client.ContainerCreate(ctx,
		&container.Config{
			Image:      imageName,
			WorkingDir: "/app",
			Cmd:        []string{"/app/app"},
			ExposedPorts: nat.PortSet{
				nat.Port(port + "/tcp"): struct{}{},
			},
		},
		&container.HostConfig{
			Binds: volumeBinds,
			// ADD PORT BINDINGS BACK for direct access
			PortBindings: nat.PortMap{
				nat.Port(port + "/tcp"): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "0", // Docker assigns random port
					},
				},
			},
			RestartPolicy: container.RestartPolicy{
				Name: "unless-stopped",
			},
		},
		nil, nil, containerName)

	if err != nil {
		logger.Error("Failed to create deployment container", err)
		return "", err
	}

	logger.Debug("✅ Successfully created deployment container", zap.String("container_id", resp.ID))
	return resp.ID, nil
}

func (c *DockerClient) ListContainers(ctx context.Context) error {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		All: true, // Include stopped containers
	})
	if err != nil {
		logger.Error("Failed to list containers", err)
		return err
	}

	if len(containers) == 0 {
		logger.Debug("No containers found")
		return nil
	}

	for _, ctr := range containers {
		logger.Debug("Container",
			zap.String("id", ctr.ID[:12]),
			zap.String("image", ctr.Image),
			zap.String("status", ctr.Status))
	}

	return nil
}
func (c *DockerClient) StartContainer(ctx context.Context, containerId string) error {
	err := c.client.ContainerStart(ctx, containerId, container.StartOptions{})
	if err != nil {
		logger.Error("failed to Start docker container", err, zap.String("container_id", containerId))
		return err
	}
	logger.Debug("Successfully Started Container", zap.String("container_id", containerId))
	return nil
}

func (c *DockerClient) StopContainer(ctx context.Context, containerId string) error {
	err := c.client.ContainerStop(ctx, containerId, container.StopOptions{})
	if err != nil {
		logger.Error("failed to Stop docker container", err, zap.String("container_id", containerId))
		return err
	}
	logger.Debug("Successfully Stopped Container", zap.String("container_id", containerId))

	return nil
}

func (c *DockerClient) InspectContainer(ctx context.Context, containerID string) (*container.InspectResponse, error) {
	resp, err := c.client.ContainerInspect(ctx, containerID)
	if err != nil {
		logger.Error("Failed to inspect container", err)
		return nil, err
	}
	logger.Debug("Successfully Inspected container", zap.String("container_id", containerID))
	return &resp, nil
}

func (c *DockerClient) WaitContainer(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return c.client.ContainerWait(ctx, containerID, condition)
}
func (c *DockerClient) DoesContainerExist(ctx context.Context, containerName string) bool {
	_, err := c.client.ContainerInspect(ctx, containerName)
	if err != nil {
		// client.IsErrNotFound is the standard way to check if the error
		// specifically means the container is missing.
		if client.IsErrNotFound(err) {
			return false
		}
		// If there's a different error (like connection issues),
		// we log it but assume it doesn't exist or is unreachable.
		logger.Error("Error inspecting container", nil, zap.Error(err))
		return false
	}

	// If err is nil, the container exists (regardless of state)
	return true
}

// RemoveContainer handles both Name and ID identifiers
func (c *DockerClient) RemoveContainer(ctx context.Context, identifier string) error {
	// If identifier is empty, skip to avoid Docker API errors
	if identifier == "" {
		return nil
	}

	logger.Debug("Removing container", zap.String("identifier", identifier))

	// Force: true is good for workers because it handles "Running" or "Exited" states
	err := c.client.ContainerRemove(ctx, identifier, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true, // Recommended: cleans up anonymous volumes too
	})

	if err != nil {
		// If the container was already deleted by something else, don't treat it as a fatal error
		if client.IsErrNotFound(err) {
			logger.Debug("Container already gone", zap.String("identifier", identifier))
			return nil
		}
		logger.Error("Failed to remove container", nil, zap.Error(err), zap.String("identifier", identifier))
		return err
	}

	logger.Debug("Successfully removed container", zap.String("identifier", identifier))
	return nil
}

func (c *DockerClient) GetContainerLogs(ctx context.Context, containerID string) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false,
		Timestamps: false,
	}
	reader, err := c.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		logger.Error("failed to get container logs", err, zap.String("container_id", containerID))
		return "", err
	}
	buf := new(strings.Builder)
	_, err = io.Copy(buf, reader)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
