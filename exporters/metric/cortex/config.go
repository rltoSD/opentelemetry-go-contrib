package cortex

import "net/http"

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
