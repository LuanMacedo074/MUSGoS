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
		"QUEUE_TYPE", "QUEUE_REDIS_HOST", "QUEUE_REDIS_PORT", "QUEUE_REDIS_PASSWORD",
		"QUEUE_REDIS_DB", "QUEUE_REDIS_KEY_PREFIX",
		"RABBITMQ_HOST", "RABBITMQ_PORT", "RABBITMQ_USER", "RABBITMQ_PASSWORD",
		"RABBITMQ_VHOST", "RABBITMQ_EXCHANGE",
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
		"EncryptionKey":    {cfg.EncryptionKey, "IPAddress resolution"},
		"Protocol":         {cfg.Protocol, "smus"},
		"SessionStoreType": {cfg.SessionStoreType, "memory"},
		"RedisHost":        {cfg.Redis.Host, "localhost"},
		"RedisPort":        {cfg.Redis.Port, "6379"},
		"RedisPassword":    {cfg.Redis.Password, ""},
		"RedisDB":          {cfg.Redis.DB, "0"},
		"RedisKeyPrefix":   {cfg.Redis.KeyPrefix, "musgo"},
		"RedisConnTTL":         {cfg.Redis.ConnTTL, "3600"},
		"QueueType":            {cfg.QueueType, "memory"},
		"QueueRedisHost":       {cfg.QueueRedis.Host, "localhost"},
		"QueueRedisPort":       {cfg.QueueRedis.Port, "6379"},
		"QueueRedisPassword":   {cfg.QueueRedis.Password, ""},
		"QueueRedisDB":         {cfg.QueueRedis.DB, "1"},
		"QueueRedisKeyPrefix":  {cfg.QueueRedis.KeyPrefix, "musgoq"},
		"RabbitMQHost":         {cfg.RabbitMQ.Host, "localhost"},
		"RabbitMQPort":         {cfg.RabbitMQ.Port, "5672"},
		"RabbitMQUser":         {cfg.RabbitMQ.User, "guest"},
		"RabbitMQPassword":     {cfg.RabbitMQ.Password, "guest"},
		"RabbitMQVHost":        {cfg.RabbitMQ.VHost, "/"},
		"RabbitMQExchange":     {cfg.RabbitMQ.Exchange, "musgo"},
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
	t.Setenv("QUEUE_TYPE", "rabbitmq")
	t.Setenv("QUEUE_REDIS_HOST", "qredis.local")
	t.Setenv("QUEUE_REDIS_PORT", "6381")
	t.Setenv("QUEUE_REDIS_PASSWORD", "qpass")
	t.Setenv("QUEUE_REDIS_DB", "3")
	t.Setenv("QUEUE_REDIS_KEY_PREFIX", "qprefix")
	t.Setenv("RABBITMQ_HOST", "rabbit.local")
	t.Setenv("RABBITMQ_PORT", "5673")
	t.Setenv("RABBITMQ_USER", "admin")
	t.Setenv("RABBITMQ_PASSWORD", "rabbitpass")
	t.Setenv("RABBITMQ_VHOST", "/prod")
	t.Setenv("RABBITMQ_EXCHANGE", "myexchange")

	cfg := config.LoadServerConfig()

	checks := map[string]struct{ got, want string }{
		"ApplicationName":    {cfg.ApplicationName, "TestApp"},
		"Port":               {cfg.Port, "9999"},
		"LogLevel":           {cfg.LogLevel, "DEBUG"},
		"LoggerType":         {cfg.LoggerType, "console"},
		"LogPath":            {cfg.LogPath, "/tmp/logs"},
		"Environment":        {cfg.Environment, "production"},
		"CipherType":         {cfg.CipherType, "aes"},
		"EncryptionKey":      {cfg.EncryptionKey, "secret123"},
		"Protocol":           {cfg.Protocol, "http"},
		"SessionStoreType":   {cfg.SessionStoreType, "redis"},
		"RedisHost":          {cfg.Redis.Host, "redis.local"},
		"RedisPort":          {cfg.Redis.Port, "6380"},
		"RedisPassword":      {cfg.Redis.Password, "s3cret"},
		"RedisDB":            {cfg.Redis.DB, "2"},
		"RedisKeyPrefix":     {cfg.Redis.KeyPrefix, "test"},
		"RedisConnTTL":       {cfg.Redis.ConnTTL, "7200"},
		"QueueType":          {cfg.QueueType, "rabbitmq"},
		"QueueRedisHost":     {cfg.QueueRedis.Host, "qredis.local"},
		"QueueRedisPort":     {cfg.QueueRedis.Port, "6381"},
		"QueueRedisPassword": {cfg.QueueRedis.Password, "qpass"},
		"QueueRedisDB":       {cfg.QueueRedis.DB, "3"},
		"QueueRedisKeyPrefix":{cfg.QueueRedis.KeyPrefix, "qprefix"},
		"RabbitMQHost":       {cfg.RabbitMQ.Host, "rabbit.local"},
		"RabbitMQPort":       {cfg.RabbitMQ.Port, "5673"},
		"RabbitMQUser":       {cfg.RabbitMQ.User, "admin"},
		"RabbitMQPassword":   {cfg.RabbitMQ.Password, "rabbitpass"},
		"RabbitMQVHost":      {cfg.RabbitMQ.VHost, "/prod"},
		"RabbitMQExchange":   {cfg.RabbitMQ.Exchange, "myexchange"},
	}

	for field, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", field, c.got, c.want)
		}
	}
}

func TestLoadServerConfig_CommandLevels_Defaults(t *testing.T) {
	cfg := config.LoadServerConfig()

	if cfg.CommandLevels == nil {
		t.Fatal("CommandLevels should not be nil")
	}
	if cfg.CommandLevels["system.user.delete"] != 80 {
		t.Errorf("system.user.delete = %d, want 80", cfg.CommandLevels["system.user.delete"])
	}
	if cfg.CommandLevels["system.server.getVersion"] != 20 {
		t.Errorf("system.server.getVersion = %d, want 20", cfg.CommandLevels["system.server.getVersion"])
	}
	if cfg.CommandLevels["DBAdmin.createUser"] != 80 {
		t.Errorf("DBAdmin.createUser = %d, want 80", cfg.CommandLevels["DBAdmin.createUser"])
	}
}

func TestLoadServerConfig_CommandLevels_EnvOverride(t *testing.T) {
	t.Setenv("USERLEVEL_SYSTEM_USER_DELETE", "100")
	t.Setenv("USERLEVEL_SYSTEM_SERVER_GETVERSION", "0")

	cfg := config.LoadServerConfig()

	if cfg.CommandLevels["system.user.delete"] != 100 {
		t.Errorf("system.user.delete = %d, want 100", cfg.CommandLevels["system.user.delete"])
	}
	if cfg.CommandLevels["system.server.getVersion"] != 0 {
		t.Errorf("system.server.getVersion = %d, want 0", cfg.CommandLevels["system.server.getVersion"])
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
