package config

import (
	"fmt"
	"strings"

	"github.com/godatei/datei/internal/buildconfig"
	"github.com/spf13/viper"
)

var v = viper.NewWithOptions()

func NewConfig(path string) error {
	v = viper.NewWithOptions(
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_")),
	)

	v.SetEnvPrefix("DATEI")
	v.AutomaticEnv()

	v.AddConfigPath(".")
	v.AddConfigPath("/var/datei")
	v.AddConfigPath("$HOME/.datei")
	v.AddConfigPath("$XDG_CONFIG_HOME/datei")
	v.SetConfigName("config")

	v.SetDefault("database.migrations", true)
	v.SetDefault("server.addr", "0.0.0.0:8080")
	if buildconfig.IsDevelopment() {
		v.SetDefault("logging.level", "debug")
	} else {
		v.SetDefault("logging.level", "info")
	}
	v.SetDefault("storage.s3.bucket", "")
	v.SetDefault("storage.s3.create_bucket", "true")
	v.SetDefault("storage.s3.endpoint", "")
	v.SetDefault("storage.s3.region", "")
	v.SetDefault("storage.s3.use_path_style", "")
	v.SetDefault("storage.s3.access_key_id", "")
	v.SetDefault("storage.s3.secret_access_key", "")
	v.SetDefault("eventstore.snapshot_threshold", 100)
	v.SetDefault("watermill.topic", "datei-events")

	if path != "" {
		v.SetConfigFile(path)
	}

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	return nil
}

func DatabaseURI() string {
	return v.GetString("database.uri")
}

func DatabaseMigrations() bool {
	return v.GetBool("database.migrations")
}

func ServerAddr() string {
	return v.GetString("server.addr")
}

func LoggingLevel() string {
	return v.GetString("logging.level")
}

type S3Config struct {
	Bucket       string
	CreateBucket bool `mapstructure:"create_bucket"`

	Endpoint        string
	Region          string
	UsePathStyle    bool   `mapstructure:"use_path_style"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

func StorageS3() (S3Config, error) {
	cfg := S3Config{CreateBucket: true}
	err := v.Sub("storage.s3").Unmarshal(&cfg)
	if err != nil {
		err = fmt.Errorf("failed to parse StorageConfig: %w", err)
	}
	return cfg, err
}

type EventStoreConfig struct {
	SnapshotThreshold int
}

func EventStore() EventStoreConfig {
	return EventStoreConfig{
		SnapshotThreshold: v.GetInt("eventstore.snapshot_threshold"),
	}
}

type WatermillConfig struct {
	Topic string
}

func Watermill() WatermillConfig {
	return WatermillConfig{
		Topic: v.GetString("watermill.topic"),
	}
}
