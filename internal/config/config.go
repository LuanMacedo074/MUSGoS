package config

import "os"

type ServerConfig struct {
	ApplicationName string
	Port            string
	LogLevel        string
	LoggerType      string
	LogPath         string
	Environment     string
	CipherType      string
	EncryptionKey   string
	Protocol        string
	DatabaseType    string
	DatabasePath    string
}

func LoadServerConfig() ServerConfig {
	return ServerConfig{
		ApplicationName: getEnv("APPLICATION_NAME", "SMUS-SERVER"),
		Port:            getEnv("PORT", "1199"),
		LogLevel:      getEnv("LOG_LEVEL", "INFO"),
		LoggerType:    getEnv("LOGGER_TYPE", "file"),
		LogPath:       getEnv("LOG_PATH", "logs"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		CipherType:    getEnv("CIPHER_TYPE", "blowfish"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "NO_ENCRYPTION_KEY"),
		Protocol:      getEnv("PROTOCOL", "smus"),
		DatabaseType:  getEnv("DATABASE_TYPE", "sqlite"),
		DatabasePath:  getEnv("DATABASE_PATH", "data/musgo.db"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
