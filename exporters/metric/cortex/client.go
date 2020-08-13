// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cortex

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	// ErrNoBasicAuthUsername occurs when no username was provided for basic
	// authentication.
	ErrNoBasicAuthUsername = fmt.Errorf("No username provided for basic authentication")

	// ErrNoBasicAuthPassword occurs when no password or password file was provided for
	// basic authentication.
	ErrNoBasicAuthPassword = fmt.Errorf("No password or password file provided for basic authentication")

	// ErrFailedToReadBasicAuthPasswordFile occurs when a password file for basic
	// authentication exists, but could not be read.
	ErrFailedToReadBasicAuthPasswordFile = fmt.Errorf("Failed to read password file for basic authentication")
)

// buildClient returns a http client that adds Authorization headers to http requests sent
// through it and uses TLS.
func (e *Exporter) buildClient() (*http.Client, error) {
	secureTransport := &SecureTransport{
		basicAuth:       e.config.BasicAuth,
		bearerToken:     e.config.BearerToken,
		bearerTokenFile: e.config.BearerTokenFile,
	}
	secureClient := http.Client{
		Transport: secureTransport,
		Timeout:   e.config.RemoteTimeout,
	}
	return &secureClient, nil
}

// SecureTransport implements http.RoundTripper. It is a custom http.Transport that
// authenticates the request by adding Authorization headers.
type SecureTransport struct {
	basicAuth       map[string]string
	bearerToken     string
	bearerTokenFile string
	rt              http.RoundTripper
}

// RoundTrip intercepts http requests and adds Authorization headers using the basic
// authentication or bearer tokens if they are provided by the user.
func (t *SecureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request since RoundTrip should not modify it.
	reqContext := req.Context()
	clonedReq := req.Clone(reqContext)

	// Set basic authentication if the user provided it.
	if err := t.addBasicAuth(clonedReq); err != nil {
		return nil, err
	}
	return t.rt.RoundTrip(clonedReq)
}

func (t *SecureTransport) addBasicAuth(req *http.Request) error {
	if t.basicAuth == nil {
		return nil
	}

	// There must be an username for basic authentication.
	username := t.basicAuth["username"]
	if username == "" {
		return fmt.Errorf("No username provided for basic authentication")
	}

	// Use password from password file if it exists.
	passwordFile := t.basicAuth["password_file"]
	if passwordFile != "" {
		file, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			return ErrFailedToReadBasicAuthPasswordFile
		}
		password := string(file)
		req.SetBasicAuth(username, password)
		return nil
	}

	// Use provided password.
	password := t.basicAuth["password"]
	if password == "" {
		return ErrNoBasicAuthPassword
	}
	req.SetBasicAuth(username, password)

	return nil
}
