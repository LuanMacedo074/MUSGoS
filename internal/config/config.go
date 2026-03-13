package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Host      string
	Port      string
	Password  string
	DB        string
	KeyPrefix string
	ConnTTL   string
}

type RabbitMQConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	VHost    string
	Exchange string
}

type ServerConfig struct {
	ApplicationName  string
	Port             string
	ServerIP         string
	MaxMessageSize   int
	TCPNoDelay       bool
	DefaultUserLevel int
	LogLevel         string
	LoggerType       string
	LogPath          string
	Environment      string
	CipherType       string
	EncryptionKey    string
	Protocol         string
	DatabaseType     string
	DatabasePath     string
	SessionStoreType string
	ScriptsPath      string
	ScriptTimeout    int
	AuthMode         string
	Redis            RedisConfig
	QueueType        string
	QueueRedis       RedisConfig
	RabbitMQ         RabbitMQConfig
}

func LoadServerConfig() ServerConfig {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: error loading .env file: %v", err)
	}

	return ServerConfig{
		ApplicationName: getEnv("APPLICATION_NAME", "SMUS-SERVER"),
		Port:            getEnv("PORT", "1199"),
		ServerIP:        getEnv("SERVER_IP", ""),
		MaxMessageSize:  getEnvInt("MAX_MESSAGE_SIZE", 8192),
		TCPNoDelay:      getEnv("TCP_NO_DELAY", "1") == "1",
		DefaultUserLevel: getEnvInt("DEFAULT_USER_LEVEL", 20),
		LogLevel:      getEnv("LOG_LEVEL", "INFO"),
		LoggerType:    getEnv("LOGGER_TYPE", "file"),
		LogPath:       getEnv("LOG_PATH", "logs"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		CipherType:    getEnv("CIPHER_TYPE", "blowfish"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "IPAddress resolution"),
		Protocol:      getEnv("PROTOCOL", "smus"),
		DatabaseType:     getEnv("DATABASE_TYPE", "sqlite"),
		DatabasePath:     getEnv("DATABASE_PATH", "data/musgo.db"),
		SessionStoreType: getEnv("SESSION_STORE_TYPE", "memory"),
		ScriptsPath:      getEnv("SCRIPTS_PATH", "external/scripts"),
		ScriptTimeout:    getEnvInt("SCRIPT_TIMEOUT", 5),
		AuthMode:         getEnv("AUTH_MODE", "open"),
		Redis: RedisConfig{
			Host:      getEnv("REDIS_HOST", "localhost"),
			Port:      getEnv("REDIS_PORT", "6379"),
			Password:  getEnv("REDIS_PASSWORD", ""),
			DB:        getEnv("REDIS_DB", "0"),
			KeyPrefix: getEnv("REDIS_KEY_PREFIX", "musgo"),
			ConnTTL:   getEnv("REDIS_CONN_TTL", "3600"),
		},
		QueueType: getEnv("QUEUE_TYPE", "memory"),
		QueueRedis: RedisConfig{
			Host:      getEnv("QUEUE_REDIS_HOST", "localhost"),
			Port:      getEnv("QUEUE_REDIS_PORT", "6379"),
			Password:  getEnv("QUEUE_REDIS_PASSWORD", ""),
			DB:        getEnv("QUEUE_REDIS_DB", "1"),
			KeyPrefix: getEnv("QUEUE_REDIS_KEY_PREFIX", "musgoq"),
		},
		RabbitMQ: RabbitMQConfig{
			Host:     getEnv("RABBITMQ_HOST", "localhost"),
			Port:     getEnv("RABBITMQ_PORT", "5672"),
			User:     getEnv("RABBITMQ_USER", "guest"),
			Password: getEnv("RABBITMQ_PASSWORD", "guest"),
			VHost:    getEnv("RABBITMQ_VHOST", "/"),
			Exchange: getEnv("RABBITMQ_EXCHANGE", "musgo"),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if n, err := strconv.Atoi(value); err == nil {
			return n
		}
	}
	return fallback
}
