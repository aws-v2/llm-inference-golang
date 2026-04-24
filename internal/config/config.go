package config

import (
	"log"
	"os"
	"strconv"
	"time"

)

type Config struct {
	AppEnv       string
	ServerPort   string
	NatsURL      string
	NatsUser     string
	NatsPassword string
	NatsPrefix   string
	DocsPath     string

	DB      DBConfig
	Eureka  EurekaConfig
	ServiceName string
	Profile     string
	Region      string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type EurekaConfig struct {
	ServerURL         string
	AppName           string
	HostName          string
	IPAddr            string
	Port              int
	VipAddress        string
	InstanceID        string
	HeartbeatInterval time.Duration
}

func Load() *Config {
	// Load .env if it exists
 

	cfg := &Config{
		AppEnv:       getEnv("APP_ENV", "dev"),
		ServerPort:   getEnv("SERVER_PORT", ":8891"),
		NatsURL:      getEnv("NATS_URL", "nats://localhost:4222"),
		NatsUser:     getEnv("NATS_USER", "auth-server"),
		NatsPassword: getEnv("NATS_PASSWORD", "auth-secret"),
		NatsPrefix:   getEnv("NATS_PREFIX", "dev.v1"),
		DocsPath:     getEnv("DOCS_PATH", "./docs"),

		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", "root"),
			Name:     getEnv("DB_NAME", "sagemaker_inference_db"),
		},
	}

	cfg.Eureka = EurekaConfig{
		ServerURL:         getEnv("EUREKA_SERVER_URL", "http://localhost:8761/eureka"),
		AppName:           getEnv("EUREKA_APP_NAME", "llm-gateway"),
		HostName:          getEnv("EUREKA_HOSTNAME", "localhost"),
		IPAddr:            getEnv("EUREKA_IP_ADDR", "127.0.0.1"),
		Port:              getEnvInt("SERVICE_PORT", 8891),
		VipAddress:        getEnv("EUREKA_VIP_ADDRESS", "llm-gateway"),
		InstanceID:        getEnv("EUREKA_INSTANCE_ID", "localhost:8891"),
		HeartbeatInterval: 30 * time.Second,
	}
	cfg.ServiceName = getEnv("SERVICE_NAME", "llm-inference-service")
	cfg.Profile = getEnv("APP_PROFILE", "dev")
	cfg.Region = getEnv("AWS_REGION", "us-east-1")
 
	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if valStr == "" {
		return fallback
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		log.Fatalf("Invalid int for %s", key)
	}
	return val
}