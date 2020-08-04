package cortex

import (
	"fmt"
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

// WithFilepath adds a path where Viper will search for the YAML file in.
func WithFilepath(filepath string) Option {
	return filepathOption(filepath)
}

type filepathOption string

func (o filepathOption) Apply(config *Config) {
	viper.AddConfigPath(string(o))
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
	if err := ValidateConfig(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// ErrNoEndpoint is an error for when the YAML file does not contain the `url` property.
var ErrNoEndpoint = fmt.Errorf("No endpoint URL found in the YAML file")

// ErrTwoPasswords is an error for when the YAML file contains both `password` and `password_file`.
var ErrTwoPasswords = fmt.Errorf("Cannot have two passwords in the YAML file")

// ErrTwoBearerTokens is an error for when the YAML file contains both `bearer_token` and
// `bearer_token_file`.
var ErrTwoBearerTokens = fmt.Errorf("Cannot have two bearer tokens in the YAML file")

// ValidateConfig checks a Config struct for missing required properties and property conflicts.
// Additionally, it adds default values to missing properties when there is a default.
func ValidateConfig(config *Config) error {
	if config.Endpoint == "" {
		return ErrNoEndpoint
	}
	if config.BearerToken != "" && config.BearerTokenFile != "" {
		return ErrTwoBearerTokens
	}
	if config.BasicAuth["password"] != "" && config.BasicAuth["password_file"] != "" {
		return ErrTwoPasswords
	}
	if config.RemoteTimeout == "" {
		config.RemoteTimeout = "30s"
	}
	return nil
}
