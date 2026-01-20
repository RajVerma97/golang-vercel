package services

import (
	"context"

	docker_client "github.com/RajVerma97/golang-vercel/backend/internal/client/docker"
	"github.com/RajVerma97/golang-vercel/backend/internal/config"
	"github.com/RajVerma97/golang-vercel/backend/internal/logger"
)

type Services struct {
	BuildService            *BuildService
	DeployService           *DeployService
	WorkspaceManagerService *WorkspaceManagerService
	GitService              *GitService
	RedisService            *RedisService
}

func NewServices(ctx context.Context, config *config.Config) (*Services, error) {
	dockerClient, err := docker_client.NewDockerClient()
	if err != nil {
		logger.Error("failed to init docker client", err)
		return nil, err
	}

	// redisClient, err := redis_client.NewRedisClient(ctx, config.Redis)
	// if err != nil {
	// 	logger.Error("failed to init redis client", err)
	// 	return nil, err
	// }
	buildService := NewBuildService(&BuildServiceConfig{
		DockerClient: dockerClient,
	})
	deployService := NewDeployService(&DeployServiceConfig{
		DockerClient: dockerClient,
	})
	workspaceManagerService := NewWorkspaceManagerService(&WorkspaceManagerServiceConfig{})
	gitService := NewGitService(&GitServiceConfig{})

	redisService := NewRedisService(&RedisServiceConfig{
		// RedisClient: redisClient,
	})

	return &Services{
		BuildService:            buildService,
		DeployService:           deployService,
		WorkspaceManagerService: workspaceManagerService,
		GitService:              gitService,
		RedisService:            redisService,
	}, nil
}
