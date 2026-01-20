package config

import "github.com/RajVerma97/golang-vercel/backend/internal/helpers"

type ServerConfig struct {
	Host string
	Port int
}
type RedisConfig struct {
	Host string
	Port int
}

type Config struct {
	Server *ServerConfig
	Redis  *RedisConfig
}

func NewConfig() *Config {
	return &Config{
		Server: &ServerConfig{
			Host: helpers.GetEnv("SERVER_HOST", ""),
			Port: helpers.GetEnv("SERVER_PORT", 0),
		},
		Redis: &RedisConfig{
			Host: helpers.GetEnv("REDIS_HOST", ""),
			Port: helpers.GetEnv("REDIS_PORT", 0),
		},
	}
}
