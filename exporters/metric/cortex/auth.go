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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

var (
	// ErrNoBasicAuthUsername occurs when no username was provided for basic
	// authentication.
	ErrNoBasicAuthUsername = fmt.Errorf("No username provided for basic authentication")

	// ErrNoBasicAuthPassword occurs when no password or password file was provided for
	// basic authentication.
	ErrNoBasicAuthPassword = fmt.Errorf("No password or password file provided for basic authentication")

	// ErrFailedToReadFile occurs when a password / bearer token file exists, but could
	// not be read.
	ErrFailedToReadFile = fmt.Errorf("Failed to read password / bearer token file")
)

// addBasicAuth sets the Authorization header for basic authentication using a username
// and a password / password file. To prevent the Exporter from potentially opening a
// password file on every request by calling this method, the Authorization header is also
// added to the Config header map.
func (e *Exporter) addBasicAuth(req *http.Request) error {
	// No need to add basic auth if it isn't provided or if the Authorization header is
	// already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}
	if e.config.BasicAuth == nil {
		return nil
	}

	// There must be an username for basic authentication.
	username := e.config.BasicAuth["username"]
	if username == "" {
		return ErrNoBasicAuthUsername
	}

	// Use password from password file if it exists.
	passwordFile := e.config.BasicAuth["password_file"]
	if passwordFile != "" {
		file, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			return ErrFailedToReadFile
		}
		password := string(file)
		req.SetBasicAuth(username, password)
		e.storeAuthHeader(req.Header.Get("Authorization"))
		return nil
	}

	// Use provided password.
	password := e.config.BasicAuth["password"]
	if password == "" {
		return ErrNoBasicAuthPassword
	}
	req.SetBasicAuth(username, password)
	e.storeAuthHeader(req.Header.Get("Authorization"))

	return nil
}

// addBearerTokenAuth sets the Authorization header for bearer tokens using a bearer token
// string or a bearer token file. To prevent the Exporter from potentially opening a
// bearer token file on every request by calling this method, the Authorization header is
// also added to the Config header map.
func (e *Exporter) addBearerTokenAuth(req *http.Request) error {
	// No need to add bearer token auth if the Authorization header is already set.
	if _, exists := e.config.Headers["Authorization"]; exists {
		return nil
	}

	// Use bearer token from bearer token file if it exists.
	if e.config.BearerTokenFile != "" {
		file, err := ioutil.ReadFile(e.config.BearerTokenFile)
		if err != nil {
			return ErrFailedToReadFile
		}
		bearerTokenString := "Bearer " + string(file)
		req.Header.Set("Authorization", bearerTokenString)
		e.storeAuthHeader(bearerTokenString)
		return nil
	}

	// Otherwise, use bearer token field.
	if e.config.BearerToken != "" {
		bearerTokenString := "Bearer " + e.config.BearerToken
		req.Header.Set("Authorization", bearerTokenString)
		e.storeAuthHeader(bearerTokenString)
	}

	return nil
}

// buildClient returns a http client that uses TLS and has the user-specified proxy and
// timeout.
func (e *Exporter) buildClient() (*http.Client, error) {
	// Create a TLS Config struct for use in a custom HTTP Transport.
	tlsConfig, err := e.buildTLSConfig()
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Convert proxy url to proxy function for use in custom Transport.
	if e.config.ProxyURL != "" {
		proxyURL, err := url.Parse(e.config.ProxyURL)
		if err != nil {
			return nil, err
		}
		proxy := http.ProxyURL(proxyURL)
		transport.Proxy = proxy
	}

	// Create and return a client that
	client := http.Client{
		Transport: transport,
		Timeout:   e.config.RemoteTimeout,
	}
	return &client, nil
}

// buildTLSConfig uses the TLSConfig map in Config to create a tls.Config struct.
func (e *Exporter) buildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if e.config.TLSConfig == nil {
		return tlsConfig, nil
	}

	// Set the server name if it exists.
	if e.config.TLSConfig["server_name"] != "" {
		tlsConfig.ServerName = e.config.TLSConfig["server_name"]
	}

	// Set InsecureSkipVerify. Viper reads the bool as a string since it is in a map.
	if e.config.TLSConfig["insecure_skip_verify"] == "1" {
		tlsConfig.InsecureSkipVerify = true
	} else {
		tlsConfig.InsecureSkipVerify = false
	}

	// Load certificates from CA file if it exists.
	if err := e.loadCACertificates(tlsConfig); err != nil {
		return nil, err
	}

	// Load the client certificate if it exists.
	if err := e.loadClientCertificate(tlsConfig); err != nil {
		return nil, err
	}

	return tlsConfig, nil
}

// loadCACertificates reads a CA file and updates the certificate pool in a tls Config
// struct.
func (e *Exporter) loadCACertificates(tlsConfig *tls.Config) error {
	caFile := e.config.TLSConfig["ca_file"]
	if caFile != "" {
		caFileData, err := ioutil.ReadFile(caFile)
		if err != nil {
			return err
		}
		certPool := x509.NewCertPool()
		certPool.AppendCertsFromPEM(caFileData)
		tlsConfig.RootCAs = certPool
	}
	return nil
}

// loadClientCertificate reads a certificate file and key and stores it in a tls Config
// struct.
func (e *Exporter) loadClientCertificate(tlsConfig *tls.Config) error {
	certFile := e.config.TLSConfig["cert_file"]
	keyFile := e.config.TLSConfig["key_file"]
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	tlsConfig.Certificates = []tls.Certificate{cert}
	return nil
}

// storeAuthHeader creates a new Headers map in the Config if it is nil and stores the
// Authorization header value in the map.
func (e *Exporter) storeAuthHeader(value string) {
	if e.config.Headers == nil {
		e.config.Headers = make(map[string]string)
	}
	e.config.Headers["Authorization"] = value
}
