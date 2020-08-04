package cortex

import (
	"net/http"

	"github.com/spf13/viper"
)

// Config contains properties the Exporter uses to export metrics data to Cortex.
type Config struct {
	Endpoint        string            `mapstructure:"url"`
	RemoteTimeout   string            `mapstructure:"remote_timeout"`
	Name            string            `mapstructure:"name"`
	BasicAuth       map[string]string `mapstructure:"basic_auth"`
	BearerToken     string            `mapstructure:"bearer_token"`
	BearerTokenFile string            `mapstructure:"bearer_token_file"`
	TLSConfig       map[string]string `mapstructure:"tls_config"`
	ProxyURL        string            `mapstructure:"proxy_url"`
	Client          *http.Client
}

// Option sets an option for a Config struct.
type Option interface {
	Apply(*Config)
}

// NewConfig creates a Config struct with a YAML file and applies Option functions to the Config
// struct.
func NewConfig(filename string, opts ...Option) (*Config, error) {
	var config Config
	for _, opt := range opts {
		opt.Apply(&config)
	}

	viper.SetConfigName(filename)
	viper.SetConfigType("yaml")
	viper.SetConfigName(filename)
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	return &config, nil
}
