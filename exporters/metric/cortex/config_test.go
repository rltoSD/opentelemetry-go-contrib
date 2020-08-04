package cortex

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// YAML file that produces a Config struct without errors.
var validYAML = []byte(`url: /api/prom/push
remote_timeout: 30s
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

// configValidStruct is the resulting Config struct from reading validYAML.
var validConfig = Config{
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
	ProxyURL: "",
	Client:   nil,
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
		expectedConfig *Config
		expectedError  error
	}{
		{
			"Valid Config file",
			validYAML,
			"config.yml",
			&validConfig,
			nil,
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
			config, err := NewConfig(test.fileName)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("Received error %v, wanted %v", err, test.expectedError)
			}
			if !cmp.Equal(config, test.expectedConfig) {
				t.Fatalf("Received Config %v, wanted %v", config, test.expectedConfig)
			}
		})
	}
}
