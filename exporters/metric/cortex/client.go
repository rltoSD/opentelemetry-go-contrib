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

import "net/http"

// buildClient returns a http client that adds Authorization headers to http requests sent
// through it and uses TLS.
func (e *Exporter) buildClient() (*http.Client, error) {
	secureTransport := &SecureTransport{
		basicAuth:       e.config.BasicAuth,
		bearerToken:     e.config.BearerToken,
		bearerTokenFile: e.config.BearerTokenFile,
		tlsConfig:       e.config.TLSConfig,
	}
	secureClient := http.Client{
		Transport: secureTransport,
		Timeout:   e.config.RemoteTimeout,
	}
	return &secureClient, nil
}

// SecureTransport implements http.RoundTripper. It sets up the client to use TLS and adds
// Authorization headers using the basic authentication or bearer tokens if they are
// provided by the user.
type SecureTransport struct {
	basicAuth       map[string]string
	bearerToken     string
	bearerTokenFile string
	tlsConfig       map[string]string
	rt              http.RoundTripper
}

func (t *SecureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.rt.RoundTrip(req)
}
