package utils_test

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"opentelemetry.io/contrib/exporters/metric/cortex"
	"opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

// This is an example YAML file that produces a Config struct without errors.
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
  test: header 
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
  test: header
`)

// YAML file with no endpoint. It should fail to produce a Config struct.
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
  test: header
`)

// YAML file with both password and password_file properties. It should fail to produce a Config
// struct since password and password_file are mutually exclusive.
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
headers:
  test: header
`)

// YAML file with both bearer_token and bearer_token_file properties. It should fail to produce a
// Config struct since bearer_token and bearer_token_file are mutually exclusive.
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
headers:
  test: header
`)

// ValidConfig is the resulting Config struct from reading validYAML.
var validConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	RemoteTimeout: 30 * time.Second,
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
	PushInterval: 5 * time.Second,
	Headers: map[string]string{
		"test": "header",
	},
	Client: http.DefaultClient,
}

// initYAML creates a YAML file at a given filepath in a in-memory file system.
func initYAML(yamlBytes []byte, path string) (afero.Fs, error) {
	// Create an in-memory file system.
	fs := afero.NewMemMapFs()

	// Retrieve the directory path.
	dirPath := filepath.Dir(path)

	// Create the directory and the file in the directory.
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
		testName       string
		yamlByteString []byte
		fileName       string
		directoryPath  string
		expectedConfig *cortex.Config
		expectedError  error
	}{
		{
			"Valid Config file",
			validYAML,
			"config.yml",
			"/test",
			&validConfig,
			nil,
		},
		{
			"No Timeout",
			noTimeoutYAML,
			"config.yml",
			"/test",
			&validConfig,
			nil,
		},
		{
			"No Endpoint URL",
			noEndpointYAML,
			"config.yml",
			"/test",
			&validConfig,
			nil,
		},
		{
			"Two passwords",
			twoPasswordsYAML,
			"config.yml",
			"/test",
			nil,
			cortex.ErrTwoPasswords,
		},
		{
			"Two Bearer Tokens",
			twoBearerTokensYAML,
			"config.yml",
			"/test",
			nil,
			cortex.ErrTwoBearerTokens,
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Create YAML file.
			fullPath := test.directoryPath + "/" + test.fileName
			fs, err := initYAML(test.yamlByteString, fullPath)
			require.Nil(t, err)

			// Create new Config struct from the specified YAML file with an in-memory filesystem.
			config, err := utils.NewConfig(
				test.fileName,
				utils.WithFilepath(test.directoryPath),
				utils.WithFilesystem(fs),
			)

			// Verify error and struct contents.
			require.Equal(t, err, test.expectedError)
			require.Equal(t, config, test.expectedConfig)
		})
	}
}

// TestWithFilepath tests whether NewConfig can find a YAML file that is not in the current
// directory.
func TestWithFilepath(t *testing.T) {
	tests := []struct {
		testName       string
		yamlByteString []byte
		fileName       string
		directoryPath  string
		addPath        bool
	}{
		{
			"Filepath provided, successful construction of Config",
			validYAML,
			"config.yml",
			"/success",
			true,
		},
		{
			"Filepath not provided, unsuccessful construction of Config",
			validYAML,
			"config.yml",
			"/fail",
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Create YAML file.
			fullPath := test.directoryPath + "/" + test.fileName
			fs, err := initYAML(test.yamlByteString, fullPath)
			require.Nil(t, err)

			// Create new Config struct from the specified YAML file with an in-memory filesystem.
			// If a path is added, Viper should be able to find the file and there should be no
			// error. Otherwise, an error should occur as Viper cannot find the file.
			if test.addPath {
				_, err := utils.NewConfig(
					test.fileName,
					utils.WithFilepath(test.directoryPath),
					utils.WithFilesystem(fs),
				)
				require.Nil(t, err)
			} else {
				_, err := utils.NewConfig(test.fileName, utils.WithFilesystem(fs))
				require.Error(t, err)
			}
		})
	}
}

// TestWithClient tests whether NewConfig successfully adds a HTTP client to the Config struct.
func TestWithClient(t *testing.T) {
	// Create a YAML file.
	fs, err := initYAML(validYAML, "/test/config.yml")
	require.Nil(t, err)

	// Create a new Config struct with a custom HTTP client.
	customClient := &http.Client{
		Timeout: 5 * time.Second,
	}
	config, _ := utils.NewConfig(
		"config.yml",
		utils.WithClient(customClient),
		utils.WithFilepath("/test"),
		utils.WithFilesystem(fs),
	)

	// Verify that the clients are the same.
	require.Equal(t, customClient, config.Client)
}
