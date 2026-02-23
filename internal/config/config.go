package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

var v = viper.NewWithOptions()

func NewConfig(path string) error {
	v = viper.NewWithOptions()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("DATEI")
	v.AutomaticEnv()

	v.AddConfigPath(".")
	v.AddConfigPath("/var/datei")
	v.AddConfigPath("$HOME/.datei")
	v.AddConfigPath("$XDG_CONFIG_HOME/datei")
	v.SetConfigName("config")

	v.SetDefault("database.migrations", true)
	v.SetDefault("server.addr", "0.0.0.0:8080")

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
