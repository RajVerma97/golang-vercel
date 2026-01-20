package services

import (
	"context"

	redis_client "github.com/RajVerma97/golang-vercel/backend/internal/client/redis"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
)

type RedisServiceConfig struct {
	RedisClient *redis_client.RedisClient
}

type RedisService struct {
	RedisClient *redis_client.RedisClient
}

func NewRedisService(config *RedisServiceConfig) *RedisService {
	return &RedisService{
		RedisClient: config.RedisClient,
	}
}

func (s *RedisService) EnqueueBuild(ctx context.Context) error {
	return s.RedisClient.EnqueueBuild(ctx)
}

func (s *RedisService) DequeueBuild(ctx context.Context) (*dto.Build, error) {
	return s.RedisClient.DequeueBuild(ctx)
}
