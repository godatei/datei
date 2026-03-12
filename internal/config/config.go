package config

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

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

	v.SetDefault("server.host", "http://localhost:4200")
	v.SetDefault("auth.jwt_secret", "")
	v.SetDefault("auth.token_expiration", "24h")
	v.SetDefault("auth.registration_enabled", true)
	v.SetDefault("auth.email_verification_required", false)
	v.SetDefault("auth.reset_token_duration", "1h")
	v.SetDefault("mailer.enabled", false)
	v.SetDefault("mailer.smtp.host", "localhost")
	v.SetDefault("mailer.smtp.port", 1025)
	v.SetDefault("mailer.smtp.username", "")
	v.SetDefault("mailer.smtp.password", "")
	v.SetDefault("mailer.smtp.from", "noreply@datei.local")

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

func ServerHost() string {
	return v.GetString("server.host")
}

func AuthJWTSecret() []byte {
	secret := v.GetString("auth.jwt_secret")
	if secret == "" {
		if buildconfig.IsRelease() {
			panic("auth.jwt_secret must be configured in release builds")
		}
		return []byte("dev-jwt-secret-do-not-use-in-production")
	}
	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return []byte(secret)
	}
	return decoded
}

func AuthTokenExpiration() time.Duration {
	d, err := time.ParseDuration(v.GetString("auth.token_expiration"))
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

func AuthRegistrationEnabled() bool {
	return v.GetBool("auth.registration_enabled")
}

func AuthEmailVerificationRequired() bool {
	return v.GetBool("auth.email_verification_required")
}

func AuthResetTokenDuration() time.Duration {
	d, err := time.ParseDuration(v.GetString("auth.reset_token_duration"))
	if err != nil {
		return 1 * time.Hour
	}
	return d
}

type MailerConfig struct {
	Enabled bool
	SMTP    SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func Mailer() MailerConfig {
	return MailerConfig{
		Enabled: v.GetBool("mailer.enabled"),
		SMTP: SMTPConfig{
			Host:     v.GetString("mailer.smtp.host"),
			Port:     v.GetInt("mailer.smtp.port"),
			Username: v.GetString("mailer.smtp.username"),
			Password: v.GetString("mailer.smtp.password"),
			From:     v.GetString("mailer.smtp.from"),
		},
	}
}
