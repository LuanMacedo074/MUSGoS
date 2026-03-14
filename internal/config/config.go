package config

import (
	"log"
	"os"
	"strconv"
	"strings"

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
	LogBufferSize    int
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
	AllEncrypted     bool
	QueueType        string
	QueueRedis       RedisConfig
	RabbitMQ         RabbitMQConfig
	CommandLevels    map[string]int
	IdleTimeout      int
	UDPPort          string
	CacheType        string
	CacheRedis       RedisConfig
}

var defaultCommandLevels = map[string]int{
	// Server
	"system.server.getVersion":    20,
	"system.server.getTime":       20,
	"system.server.getUserCount":  20,
	"system.server.getMovieCount": 20,
	"system.server.getMovies":     20,
	// Movie
	"system.movie.getUserCount":   20,
	"system.movie.getGroups":      20,
	"system.movie.getGroupCount":  20,
	// Group
	"system.group.join":              20,
	"system.group.leave":             20,
	"system.group.getUsers":          20,
	"system.group.getUserCount":      20,
	"system.group.setAttribute":      20,
	"system.group.getAttribute":      20,
	"system.group.deleteAttribute":   20,
	"system.group.getAttributeNames": 20,
	// User
	"system.user.getAddress":   20,
	"system.user.getGroups":    20,
	"system.user.delete":       80,
	// DBPlayer
	"DBPlayer.getAttribute":      20,
	"DBPlayer.setAttribute":      20,
	"DBPlayer.deleteAttribute":   20,
	"DBPlayer.getAttributeNames": 20,
	// DBApplication
	"DBApplication.getAttribute":      20,
	"DBApplication.setAttribute":      20,
	"DBApplication.deleteAttribute":   20,
	"DBApplication.getAttributeNames": 20,
	// DBAdmin
	"DBAdmin.createApplication": 80,
	"DBAdmin.deleteApplication": 80,
	"DBAdmin.createUser":        80,
	"DBAdmin.deleteUser":        80,
	"DBAdmin.getUserCount":      80,
	"DBAdmin.ban":               80,
	"DBAdmin.revokeBan":         80,
	// Email
	"system.server.sendEmail": 80,
	// Kill Timers
	"system.server.setKillTimer":    80,
	"system.server.cancelKillTimer": 80,
	"system.user.setKillTimer":      80,
	"system.user.cancelKillTimer":   80,
}

func LoadServerConfig() ServerConfig {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("Warning: error loading .env file: %v", err)
	}

	cfg := ServerConfig{
		ApplicationName: getEnv("APPLICATION_NAME", "SMUS-SERVER"),
		Port:            getEnv("PORT", "1199"),
		ServerIP:        getEnv("SERVER_IP", ""),
		MaxMessageSize:  getEnvInt("MAX_MESSAGE_SIZE", 8192),
		TCPNoDelay:      getEnv("TCP_NO_DELAY", "1") == "1",
		DefaultUserLevel: getEnvInt("DEFAULT_USER_LEVEL", 20),
		LogLevel:      getEnv("LOG_LEVEL", "INFO"),
		LoggerType:    getEnv("LOGGER_TYPE", "file"),
		LogPath:       getEnv("LOG_PATH", "logs"),
		LogBufferSize: getEnvInt("LOG_BUFFER_SIZE", 1024),
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
		AllEncrypted: strings.HasPrefix(getEnv("ENCRYPTION_KEY", "IPAddress resolution"), "#All"),
		QueueType:    getEnv("QUEUE_TYPE", "memory"),
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

	cfg.IdleTimeout = getEnvInt("IDLE_TIMEOUT", 0)
	cfg.UDPPort = getEnv("UDP_PORT", "")
	cfg.CacheType = getEnv("CACHE_TYPE", "memory")
	cfg.CacheRedis = RedisConfig{
		Host:      getEnv("CACHE_REDIS_HOST", "localhost"),
		Port:      getEnv("CACHE_REDIS_PORT", "6379"),
		Password:  getEnv("CACHE_REDIS_PASSWORD", ""),
		DB:        getEnv("CACHE_REDIS_DB", "2"),
		KeyPrefix: getEnv("CACHE_REDIS_KEY_PREFIX", "musgoc"),
	}
	cfg.CommandLevels = loadCommandLevels()

	return cfg
}

func loadCommandLevels() map[string]int {
	levels := make(map[string]int, len(defaultCommandLevels))
	for k, v := range defaultCommandLevels {
		levels[k] = v
	}

	// Override from env vars: USERLEVEL_SYSTEM_SERVER_GETVERSION=20
	// Conversion: env var key → dots+camelCase subject
	// Build reverse lookup: normalized subject → original subject
	lookup := make(map[string]string, len(defaultCommandLevels))
	for subject := range defaultCommandLevels {
		normalized := strings.ToUpper(strings.ReplaceAll(subject, ".", "_"))
		lookup[normalized] = subject
	}

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "USERLEVEL_") {
			continue
		}
		eqIdx := strings.IndexByte(env, '=')
		if eqIdx < 0 {
			continue
		}
		key := env[:eqIdx]
		val := env[eqIdx+1:]

		suffix := strings.TrimPrefix(key, "USERLEVEL_")
		if subject, ok := lookup[suffix]; ok {
			if n, err := strconv.Atoi(val); err == nil {
				levels[subject] = n
			}
		}
	}

	return levels
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
