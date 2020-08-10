package cortex_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/exporters/metric/cortex"
)

// Default http client with a timeout of 30 seconds.
var defaultClientWithTimeout = &http.Client{
	Timeout: 30 * time.Second,
}

// Config struct with default values. This is used to verify the output of Validate().
var ValidatedStandardConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Standard Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	Client:        defaultClientWithTimeout,
}

// Config struct with default values other than the remote timeout. This is used to verify the
// output of Validate().
var ValidatedCustomTimeoutConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Standard Config",
	RemoteTimeout: 10 * time.Second,
	PushInterval:  10 * time.Second,
	Client: &http.Client{
		Timeout: 10 * time.Second,
	},
}

// Example Config struct with a custom remote timeout.
var ExampleRemoteTimeoutConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Standard Config",
	PushInterval:  10 * time.Second,
	RemoteTimeout: 10 * time.Second,
}

// Example Config struct without a remote timeout.
var ExampleNoRemoteTimeoutConfig = cortex.Config{
	Endpoint:     "/api/prom/push",
	Name:         "Standard Config",
	PushInterval: 10 * time.Second,
}

// Example Config struct without a push interval.
var ExampleNoPushIntervalConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Standard Config",
	RemoteTimeout: 30 * time.Second,
}

// Example Config struct without a http client.
var ExampleNoClientConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Standard Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
}

// Example Config struct without an endpoint.
var ExampleNoEndpointConfig = cortex.Config{
	Name:          "Standard Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
}

// This is an example Config struct with two bearer tokens.
var ExampleTwoBearerTokenConfig = cortex.Config{
	Endpoint:        "/api/prom/push",
	Name:            "Standard Config",
	RemoteTimeout:   30 * time.Second,
	PushInterval:    10 * time.Second,
	BearerToken:     "bearer_token",
	BearerTokenFile: "bearer_token_file",
}

// This is an example Config struct with two passwords.
var ExampleTwoPasswordConfig = cortex.Config{
	Endpoint:      "/api/prom/push",
	Name:          "Standard Config",
	RemoteTimeout: 30 * time.Second,
	PushInterval:  10 * time.Second,
	BasicAuth: map[string]string{
		"username":      "user",
		"password":      "password",
		"password_file": "passwordFile",
	},
}

// TestValidate checks whether Validate() returns the correct error and sets the correct default
// values.
func TestValidate(t *testing.T) {
	tests := []struct {
		testName       string
		config         *cortex.Config
		expectedConfig *cortex.Config
		expectedError  error
	}{
		{
			testName:       "Config with Conflicting Bearer Tokens",
			config:         &ExampleTwoBearerTokenConfig,
			expectedConfig: nil,
			expectedError:  cortex.ErrTwoBearerTokens,
		},
		{
			testName:       "Config with Conflicting Passwords",
			config:         &ExampleTwoPasswordConfig,
			expectedConfig: nil,
			expectedError:  cortex.ErrTwoPasswords,
		},
		{
			testName:       "Config with Custom Timeout",
			config:         &ExampleRemoteTimeoutConfig,
			expectedConfig: &ValidatedCustomTimeoutConfig,
			expectedError:  nil,
		},
		{
			testName:       "Config with no Endpoint",
			config:         &ExampleNoEndpointConfig,
			expectedConfig: &ValidatedStandardConfig,
			expectedError:  nil,
		},
		{
			testName:       "Config with no Remote Timeout",
			config:         &ExampleNoRemoteTimeoutConfig,
			expectedConfig: &ValidatedStandardConfig,
			expectedError:  nil,
		},
		{
			testName:       "Config with no Push Interval",
			config:         &ExampleNoPushIntervalConfig,
			expectedConfig: &ValidatedStandardConfig,
			expectedError:  nil,
		},
		{
			testName:       "Config with no Client",
			config:         &ExampleNoClientConfig,
			expectedConfig: &ValidatedStandardConfig,
			expectedError:  nil,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			err := test.config.Validate()
			require.Equal(t, err, test.expectedError)
			if err == nil {
				require.Equal(t, test.config, test.expectedConfig)
			}
		})
	}
}
