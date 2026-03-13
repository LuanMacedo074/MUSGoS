package config_test

import (
	"os"
	"testing"

	"fsos-server/internal/config"
)

func TestLoadServerConfig_Defaults(t *testing.T) {
	envVars := []string{
		"APPLICATION_NAME", "PORT", "LOG_LEVEL", "LOGGER_TYPE",
		"LOG_PATH", "ENVIRONMENT", "CIPHER_TYPE", "ENCRYPTION_KEY", "PROTOCOL",
		"SESSION_STORE_TYPE", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD",
		"REDIS_DB", "REDIS_KEY_PREFIX", "REDIS_CONN_TTL",
	}
	// Save and unset all env vars, restore on cleanup
	for _, v := range envVars {
		orig, existed := os.LookupEnv(v)
		os.Unsetenv(v)
		if existed {
			t.Cleanup(func() { os.Setenv(v, orig) })
		} else {
			t.Cleanup(func() { os.Unsetenv(v) })
		}
	}

	cfg := config.LoadServerConfig()

	defaults := map[string]struct{ got, want string }{
		"ApplicationName": {cfg.ApplicationName, "SMUS-SERVER"},
		"Port":            {cfg.Port, "1199"},
		"LogLevel":        {cfg.LogLevel, "INFO"},
		"LoggerType":      {cfg.LoggerType, "file"},
		"LogPath":         {cfg.LogPath, "logs"},
		"Environment":     {cfg.Environment, "development"},
		"CipherType":      {cfg.CipherType, "blowfish"},
		"EncryptionKey":    {cfg.EncryptionKey, "NO_ENCRYPTION_KEY"},
		"Protocol":         {cfg.Protocol, "smus"},
		"SessionStoreType": {cfg.SessionStoreType, "memory"},
		"RedisHost":        {cfg.Redis.Host, "localhost"},
		"RedisPort":        {cfg.Redis.Port, "6379"},
		"RedisPassword":    {cfg.Redis.Password, ""},
		"RedisDB":          {cfg.Redis.DB, "0"},
		"RedisKeyPrefix":   {cfg.Redis.KeyPrefix, "musgo"},
		"RedisConnTTL":     {cfg.Redis.ConnTTL, "3600"},
	}
	for field, c := range defaults {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", field, c.got, c.want)
		}
	}
}

func TestLoadServerConfig_CustomValues(t *testing.T) {
	t.Setenv("APPLICATION_NAME", "TestApp")
	t.Setenv("PORT", "9999")
	t.Setenv("LOG_LEVEL", "DEBUG")
	t.Setenv("LOGGER_TYPE", "console")
	t.Setenv("LOG_PATH", "/tmp/logs")
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("CIPHER_TYPE", "aes")
	t.Setenv("ENCRYPTION_KEY", "secret123")
	t.Setenv("PROTOCOL", "http")
	t.Setenv("SESSION_STORE_TYPE", "redis")
	t.Setenv("REDIS_HOST", "redis.local")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "s3cret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_KEY_PREFIX", "test")
	t.Setenv("REDIS_CONN_TTL", "7200")

	cfg := config.LoadServerConfig()

	checks := map[string]struct{ got, want string }{
		"ApplicationName":  {cfg.ApplicationName, "TestApp"},
		"Port":             {cfg.Port, "9999"},
		"LogLevel":         {cfg.LogLevel, "DEBUG"},
		"LoggerType":       {cfg.LoggerType, "console"},
		"LogPath":          {cfg.LogPath, "/tmp/logs"},
		"Environment":      {cfg.Environment, "production"},
		"CipherType":       {cfg.CipherType, "aes"},
		"EncryptionKey":    {cfg.EncryptionKey, "secret123"},
		"Protocol":         {cfg.Protocol, "http"},
		"SessionStoreType": {cfg.SessionStoreType, "redis"},
		"RedisHost":        {cfg.Redis.Host, "redis.local"},
		"RedisPort":        {cfg.Redis.Port, "6380"},
		"RedisPassword":    {cfg.Redis.Password, "s3cret"},
		"RedisDB":          {cfg.Redis.DB, "2"},
		"RedisKeyPrefix":   {cfg.Redis.KeyPrefix, "test"},
		"RedisConnTTL":     {cfg.Redis.ConnTTL, "7200"},
	}

	for field, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", field, c.got, c.want)
		}
	}
}

func TestLoadServerConfig_PartialOverride(t *testing.T) {
	// Only override some vars, rest should be defaults
	t.Setenv("APPLICATION_NAME", "CustomApp")
	t.Setenv("PORT", "8080")

	cfg := config.LoadServerConfig()

	if cfg.ApplicationName != "CustomApp" {
		t.Errorf("ApplicationName = %q, want %q", cfg.ApplicationName, "CustomApp")
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8080")
	}
	// These should still have values (either defaults or from env)
	if cfg.LogLevel == "" {
		t.Error("LogLevel should not be empty")
	}
	if cfg.Protocol == "" {
		t.Error("Protocol should not be empty")
	}
}
