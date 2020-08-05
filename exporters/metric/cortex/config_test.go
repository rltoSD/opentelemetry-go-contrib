package cortex_test

import (
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/afero"
	"opentelemetry.io/contrib/exporters/metric/cortex"
)

// YAML file that produces a Config struct without errors.
var validYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  "X-Prometheus-Remote-Write-Version": "0.1.0"
  "tenant-ID": "123"
`)

// YAML file with no remote_timout property. It should produce a Config struct without errors.
var noTimeoutYAML = []byte(`url: /api/prom/push
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  "X-Prometheus-Remote-Write-Version": "0.1.0"
  "tenant-ID": "123"
`)

// YAML file with both password and password_file properties. It should fail to produce a Config
// struct since password and password_file are mutually exclusive.
var noEndpointYAML = []byte(`remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  "X-Prometheus-Remote-Write-Version": "0.1.0"
  "tenant-ID": "123"
`)

// YAML file with no tenant ID property. It should fail to produce a Config struct.
var noXPrometheusRemoteWriteVersionYAML = []byte(`remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  "tenant-ID": "123"
`)

// YAML file with no X-Prometheus-Remote-Write-Version. It should fail to produce a Config
// struct since X-Prometheus-Remote-Write-Version is a required header.
var noTenantIDYAML = []byte(`remote_timeout: 30s
push_interval: 5s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
headers:
  "X-Prometheus-Remote-Write-Version": "0.1.0"
`)

// YAML file with both bearer_token and bearer_token_file properties. It should fail to produce a
// Config struct since bearer_token and bearer_token_file are mutually exclusive.
var twoPasswordsYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
  password_file: passwordfile
bearer_token: qwerty12345
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)

// configValidStruct is the resulting Config struct from reading validYAML.
var twoBearerTokensYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
name: Valid Config Example
basic_auth:
  username: user
  password: password
bearer_token: qwerty12345
bearer_token_file: bearertokenfile
tls_config:
  ca_file: cafile
  cert_file: certfile
  key_file: keyfile
  server_name: server
  insecure_skip_verify: true
`)

// ValidConfig is the resulting Config struct from reading validYAML.
var ValidConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	RemoteTimeout: "30s",
	Name:          "Valid Config Example",
	BasicAuth: map[string]string{
		"username": "user",
		"password": "password",
	},
	BearerToken:     "qwerty12345",
	BearerTokenFile: "",
	TLSConfig: map[string]string{
		"ca_file":              "cafile",
		"cert_file":            "certfile",
		"key_file":             "keyfile",
		"server_name":          "server",
		"insecure_skip_verify": "1",
	},
	ProxyURL:     "",
	PushInterval: "5s",
	Headers: map[string]string{
		"x-prometheus-remote-write-version": "0.1.0",
		"tenant-id":                         "123",
	},
	Client: http.DefaultClient,
}

// initYAML creates a YAML file at a given filepath in a in-memory file system.
func initYAML(yamlBytes []byte, path string) (afero.Fs, error) {
	// Create an in-memory file system.
	fs := afero.NewMemMapFs()

	// Retrieve the directory from the filepath.
	dirPath := filepath.Dir(path)

	if err := fs.MkdirAll(dirPath, 0755); err != nil {
		return nil, err
	}
	if err := afero.WriteFile(fs, path, yamlBytes, 0644); err != nil {
		return nil, err
	}

	return fs, nil
}

// TestNewConfig tests whether NewConfig returns a correct Config struct. It checks whether the YAML
// file was read correctly and whether validation of the struct succeeded.
func TestNewConfig(t *testing.T) {
	tests := []struct {
		name           string
		yamlFile       []byte
		fileName       string
		expectedConfig *cortex.Config
		expectedError  error
	}{
		{
			"Valid Config file",
			validYAML,
			"config.yml",
			&ValidConfig,
			nil,
		},
		{
			"No Timeout",
			noTimeoutYAML,
			"config.yml",
			&ValidConfig,
			nil,
		},
		{
			"No Endpoint URL",
			noEndpointYAML,
			"config.yml",
			&ValidConfig,
			nil,
		},
		{
			"No X-Prometheus-Remote-Write-Version",
			noXPrometheusRemoteWriteVersionYAML,
			"config.yml",
			nil,
			cortex.ErrNoXPrometheusRemoteWriteVersion,
		},
		{
			"No Tenant ID",
			noTenantIDYAML,
			"config.yml",
			nil,
			cortex.ErrNoTenantID,
		},
		{
			"Two passwords",
			twoPasswordsYAML,
			"config.yml",
			nil,
			cortex.ErrTwoPasswords,
		},
		{
			"Two Bearer Tokens",
			twoBearerTokensYAML,
			"config.yml",
			nil,
			cortex.ErrTwoBearerTokens,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create YAML file.
			fs, err := initYAML(test.yamlFile, "/test/"+test.fileName)
			if err != nil {
				t.Fatalf("Failed to initialize YAML file with error %v", err)
			}

			// Create new Config struct and verify errors and contents.
			config, err := cortex.NewConfig(test.fileName, cortex.WithFilepath("/test"), cortex.WithFileSystem(fs))
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("Received error %v, wanted %v", err, test.expectedError)
			}
			if !cmp.Equal(config, test.expectedConfig) {
				t.Fatalf("Received Config %v, wanted %v", config, test.expectedConfig)
			}
		})
	}
}

// TestWithFilepath tests whether NewConfig can find a YAML file that is not in the current
// directory.
func TestWithFilepath(t *testing.T) {
	t.Run("Filepath provided", func(t *testing.T) {
		// Create YAML file.
		fs, err := initYAML(validYAML, "/test1/config.yml")
		if err != nil {
			t.Fatalf("Failed to initialize YAML file with error %v", err)
		}

		// Create new Config struct and verify that an error did not occur
		_, err = cortex.NewConfig("config.yml", cortex.WithFilepath("/test1"), cortex.WithFileSystem(fs))
		if err != nil {
			t.Fatalf("Received error '%v', wanted '%v'", err, nil)
		}
	})

	t.Run("No filepath provided", func(t *testing.T) {
		// Create YAML file.
		fs, err := initYAML(validYAML, "/test2/config.yml")
		if err != nil {
			t.Fatalf("Failed to initialize YAML file with error %v", err)
		}

		// Create new Config struct and verify that an error occurred.
		_, err = cortex.NewConfig("config.yml", cortex.WithFileSystem(fs))
		if err == nil {
			t.Fatalf("Should have failed to find YAML file")
		}

	})
}

// TestWithClient tests whether NewConfig successfully adds a HTTP client to the Config struct.
func TestWithClient(t *testing.T) {
	// Create a YAML file.
	fs, err := initYAML(validYAML, "/test/config.yml")
	if err != nil {
		t.Errorf("Failed to initialize YAML file with error %v", err)
	}

	// Create a new Config struct with a custom HTTP client.
	customClient := http.DefaultClient
	config, _ := cortex.NewConfig("config.yml", cortex.WithClient(customClient), cortex.WithFilepath("/test"), cortex.WithFileSystem(fs))

	// Verify that the clients are the same.
	if !cmp.Equal(config.Client, customClient) {
		t.Fatalf("Received client %v, wanted %v", *config.Client, *customClient)
	}
}
