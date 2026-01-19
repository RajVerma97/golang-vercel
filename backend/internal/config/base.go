package config

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
			Host: "localhost",
			Port: 8081,
		},
		Redis: &RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}
}
