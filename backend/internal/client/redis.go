package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/RajVerma97/golang-vercel/backend/internal/config"
	"github.com/RajVerma97/golang-vercel/backend/internal/dto"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	config *config.RedisConfig
	client *redis.Client
}

func NewRedisClient(config *config.RedisConfig) (*RedisClient, error) {
	redisClient := &RedisClient{
		config: config,
	}
	return redisClient, nil
}

func (c *RedisClient) Connect(ctx context.Context) error {
	fmt.Printf("CONNECTING TO RDIS ")
	redisAddr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	c.client = redis.NewClient(&redis.Options{Addr: redisAddr})

	// ping
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	fmt.Printf("\nSUCCESSFULLY CONNECTED TO REDIS %s\n", redisAddr)

	return nil
}

func (c *RedisClient) Close() error {
	// if there is already a redis existing redis connection, then only close connection
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

func (c *RedisClient) EnqueueBuild(ctx context.Context) error {
	jsonJob := dto.Job{
		ID:      1,
		RepoUrl: "github.com/demo",
		Status:  dto.JobStatusPending,
	}
	fmt.Println("ENQUEUING")
	data, err := json.Marshal(jsonJob)
	if err != nil {
		return fmt.Errorf("failed to marhsal data:%w ", err)
	}
	err = c.client.LPush(ctx, "builds", data).Err()
	if err != nil {
		return err
	}
	fmt.Println("ENQUEUED BUILD JOB SUCCESSFULLY")
	return nil
}

func (c *RedisClient) DequeueBuild(ctx context.Context) (*dto.Job, error) {
	data, err := c.client.RPop(ctx, "builds").Result()
	if err != nil {
		if err == redis.Nil {
			// Queue is empty, return nil for both without error
			return nil, nil
		}
		return nil, fmt.Errorf("failed to rpop: %w", err)
	}
	var job *dto.Job
	if err := json.Unmarshal([]byte(data), &job); err != nil {
		return nil, err
	}

	fmt.Println("DEQUEUNING")
	fmt.Println(job)
	return job, nil
}
