package config

import (
	"os"
	"strconv"
)

type Config struct {
	HTTPAddr  string
	GRPCAddr  string
	Database  DatabaseConfig
	Redis     RedisConfig
	Services  ServicesConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host string
	Port int
}

type ServicesConfig struct {
	OrderService ServiceConfig
}

type ServiceConfig struct {
	Host string
	Port int
}

func Load() (*Config, error) {
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	redisPort, _ := strconv.Atoi(getEnv("REDIS_PORT", "6379"))
	orderPort, _ := strconv.Atoi(getEnv("ORDER_SERVICE_PORT", "50051"))

	return &Config{
		HTTPAddr: getEnv("HTTP_ADDR", ":8081"),
		GRPCAddr: getEnv("GRPC_ADDR", ":50052"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "delivery_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host: getEnv("REDIS_HOST", "localhost"),
			Port: redisPort,
		},
		Services: ServicesConfig{
			OrderService: ServiceConfig{
				Host: getEnv("ORDER_SERVICE_HOST", "localhost"),
				Port: orderPort,
			},
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}