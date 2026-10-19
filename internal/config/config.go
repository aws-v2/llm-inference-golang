package config

import (
	"fmt"
	"log"
	"net"
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
	MaxPipelines int
METRICS_API_KEY string
	DB      DBConfig
	Eureka  EurekaConfig
	ServiceName string
	Profile     string
	ControlPlane string
	Region      string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode string
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
 
	ip,err :=GetOutboundIP()

	if err != nil{
		fmt.Printf("Could not get the ip address for the control plane ")
		return nil
	}

	fmt.Printf("=====,%s", ip)

	cfg := &Config{
		METRICS_API_KEY:"e956807e27669001319d73d3a9f1728a4a1161c79a6a287160c3a73394c05df8",
		ControlPlane: getEnv("CONTROL_PLANE", fmt.Sprintf("http://%s:8080",ip)),
		AppEnv:       getEnv("APP_ENV", "dev"),
		MaxPipelines:       getEnvInt("MAX_PIPELINES", 10),
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
			Name:     getEnv("DB_NAME", "llm_db"),
			SSLMode:     getEnv("DB_SSLMODE", "disable"),
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
func GetOutboundIP() (net.IP, error) {
	// 8.8.8.8:80 is just used as a routing target — no actual packet is sent for UDP Dial,
	// the OS just resolves which local interface/route would be used.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("failed to determine outbound route: %w", err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// GetOutboundInterfaceName goes one step further — tells you WHICH interface (e.g. "wlp2s0")
// owns that outbound IP, in case you need the name too.
func GetOutboundInterfaceName() (string, net.IP, error) {
	outboundIP, err := GetOutboundIP()
	if err != nil {
		return "", nil, err
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ipNet.IP.Equal(outboundIP) {
				return iface.Name, outboundIP, nil
			}
		}
	}

	return "", outboundIP, fmt.Errorf("could not match outbound IP to a named interface")
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