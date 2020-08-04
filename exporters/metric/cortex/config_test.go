package cortex_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	Client:       http.DefaultClient,
}

// initYAML creates a YAML file at a given filepath. It does not remove any created directories or
// files.
func initYAML(yamlBytes []byte, path string) error {
	dirPath := filepath.Dir(path)

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, yamlBytes, 0644)
	if err != nil {
		return err
	}

	return nil
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
			// Create YAML file in the current directory.
			if err := initYAML(test.yamlFile, test.fileName); err != nil {
				t.Fatalf("Failed to initialize YAML file with error %v", err)
			}
			defer os.RemoveAll(test.fileName)

			// Create new Config struct and verify errors and contents.
			config, err := cortex.NewConfig(test.fileName)
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
		if err := initYAML(validYAML, "./test1/config.yml"); err != nil {
			t.Fatalf("Failed to initialize YAML file with error %v", err)
		}
		defer os.RemoveAll("./test1")

		// Create new Config struct and verify that an error did not occur
		_, err := cortex.NewConfig("config.yml", cortex.WithFilepath("./test1"))
		if err != nil {
			t.Fatalf("Received error '%v', wanted '%v'", err, nil)
		}
	})

	t.Run("No filepath provided", func(t *testing.T) {
		// Create YAML file.
		if err := initYAML(validYAML, "./test2/config.yml"); err != nil {
			t.Fatalf("Failed to initialize YAML file with error %v", err)
		}
		defer os.RemoveAll("./test2")

		// Create new Config struct and verify that an error occurred.
		_, err := cortex.NewConfig("config.yml")
		if err == nil {
			t.Fatalf("Should have failed to find YAML file")
		}

	})
}

// TestWithClient tests whether NewConfig successfully adds a HTTP client to the Config struct.
func TestWithClient(t *testing.T) {
	// Create a YAML file.
	if err := initYAML(validYAML, "./config.yml"); err != nil {
		t.Errorf("Failed to initialize YAML file with error %v", err)
	}
	defer os.RemoveAll("config.yml")

	// Create a new Config struct with a custom HTTP client.
	customClient := http.DefaultClient
	config, _ := cortex.NewConfig("config.yml", cortex.WithClient(customClient))

	// Verify that the clients are the same.
	if !cmp.Equal(config.Client, customClient) {
		t.Fatalf("Received client %v, wanted %v", *config.Client, *customClient)
	}
}
