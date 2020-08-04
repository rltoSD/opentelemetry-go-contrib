package cortex

import (
	"fmt"
	"net/http"

	"github.com/spf13/viper"
)

var (
	// ErrTwoPasswords is an error for when the YAML file contains both `password` and `password_file`.
	ErrTwoPasswords = fmt.Errorf("Cannot have two passwords in the YAML file")

	// ErrTwoBearerTokens is an error for when the YAML file contains both `bearer_token` and
	// `bearer_token_file`.
	ErrTwoBearerTokens = fmt.Errorf("Cannot have two bearer tokens in the YAML file")
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
	PushInterval    string            `mapstructure:"push_interval"`
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

// WithClient adds a custom http.Client to the Config struct.
func WithClient(client *http.Client) Option {
	return clientOption{client}
}

type clientOption struct {
	client *http.Client
}

func (o clientOption) Apply(config *Config) {
	config.Client = (*http.Client)(o.client)
}

// NewConfig creates a Config struct with a YAML file and applies Option functions to the Config
// struct.
func NewConfig(filename string, opts ...Option) (*Config, error) {
	var config Config

	viper.SetConfigName(filename)
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	for _, opt := range opts {
		opt.Apply(&config)
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

// Validate checks a Config struct for missing required properties and property conflicts.
// Additionally, it adds default values to missing properties when there is a default.
func (c *Config) Validate() error {
	if c.BearerToken != "" && c.BearerTokenFile != "" {
		return ErrTwoBearerTokens
	}
	if c.BasicAuth["password"] != "" && c.BasicAuth["password_file"] != "" {
		return ErrTwoPasswords
	}
	if c.Endpoint == "" {
		c.Endpoint = "/api/prom/push"
	}
	if c.RemoteTimeout == "" {
		c.RemoteTimeout = "30s"
	}
	// Default time interval between pushes for the push controller is 10s.
	if c.PushInterval == "" {
		c.PushInterval = "10s"
	}
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	return nil
}
