package docker_client

import (
	"context"
	"io"
	"os"

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

func (c *DockerClient) CreateNewContainer(image string) (*string, error) {
	containerName := image
	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "8000",
	}
	containerPort, err := nat.NewPort("tcp", "80")
	if err != nil {
		logger.Error("failed to get the port", nil)
		return nil, nil
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}
	cont, err := c.client.ContainerCreate(context.Background(),
		&container.Config{
			Image: image,
		},
		&container.HostConfig{
			PortBindings: portBinding,
		}, nil, nil, containerName)
	if err != nil {
		logger.Error("failed to create container", err)
		panic(err)
	}

	if err := c.StartContainer(context.Background(), cont.ID); err != nil {
		return nil, err
	}
	logger.Debug("Successfully Created Docker container", zap.String("image_name", image), zap.String("container_id", cont.ID))
	return &cont.ID, nil
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

func (c *DockerClient) RemoveContainer(ctx context.Context, containerID string) error {
	logger.Debug("Removing container", zap.String("container_id", containerID))

	err := c.client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true, // Force remove even if running
	})
	if err != nil {
		logger.Error("Failed to remove container", err)
		return err
	}

	logger.Debug("Successfully removed container", zap.String("container_id", containerID))
	return nil
}
