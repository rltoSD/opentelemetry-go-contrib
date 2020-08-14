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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAuthentication checks whether http requests are properly authenticated with either
// bearer tokens or basic authentication in the addHeaders method.
func TestAuthentication(t *testing.T) {
	tests := []struct {
		testName                      string
		basicAuth                     map[string]string
		basicAuthPasswordFileContents []byte
		bearerToken                   string
		bearerTokenFile               string
		bearerTokenFileContents       []byte
		expectedAuthHeaderValue       string
		expectedError                 error
	}{
		{
			testName: "Basic Auth with password",
			basicAuth: map[string]string{
				"username": "TestUser",
				"password": "TestPassword",
			},
			expectedAuthHeaderValue: "Basic " + base64.StdEncoding.EncodeToString(
				[]byte("TestUser:TestPassword"),
			),
			expectedError: nil,
		},
		{
			testName: "Basic Auth with no username",
			basicAuth: map[string]string{
				"password": "TestPassword",
			},
			expectedAuthHeaderValue: "",
			expectedError:           ErrNoBasicAuthUsername,
		},
		{
			testName: "Basic Auth with no password",
			basicAuth: map[string]string{
				"username": "TestUser",
			},
			expectedAuthHeaderValue: "",
			expectedError:           ErrNoBasicAuthPassword,
		},
		{
			testName: "Basic Auth with password file",
			basicAuth: map[string]string{
				"username":      "TestUser",
				"password_file": "passwordFile",
			},
			basicAuthPasswordFileContents: []byte("TestPassword"),
			expectedAuthHeaderValue: "Basic " + base64.StdEncoding.EncodeToString(
				[]byte("TestUser:TestPassword"),
			),
			expectedError: nil,
		},
		{
			testName: "Basic Auth with bad password file",
			basicAuth: map[string]string{
				"username":      "TestUser",
				"password_file": "missingPasswordFile",
			},
			expectedAuthHeaderValue: "",
			expectedError:           ErrFailedToReadFile,
		},
		{
			testName:                "Bearer Token",
			bearerToken:             "testToken",
			expectedAuthHeaderValue: "Bearer testToken",
			expectedError:           nil,
		},
		{
			testName:                "Bearer Token with bad bearer token file",
			bearerTokenFile:         "missingBearerTokenFile",
			expectedAuthHeaderValue: "",
			expectedError:           ErrFailedToReadFile,
		},
		{
			testName:                "Bearer Token with bearer token file",
			bearerTokenFile:         "bearerTokenFile",
			expectedAuthHeaderValue: "Bearer testToken",
			bearerTokenFileContents: []byte("testToken"),
			expectedError:           nil,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Set up a test server that runs a handler function when it receives a http
			// request. The server writes the request's Authorization header to the
			// response body.
			handler := func(rw http.ResponseWriter, req *http.Request) {
				authHeaderValue := req.Header.Get("Authorization")
				rw.Write([]byte(authHeaderValue))
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			// Create the necessary files for tests.
			if test.basicAuth != nil {
				passwordFile := test.basicAuth["password_file"]
				if passwordFile != "" && test.basicAuthPasswordFileContents != nil {
					filepath := "./" + test.basicAuth["password_file"]
					err := createFile(test.basicAuthPasswordFileContents, filepath)
					require.Nil(t, err)
					defer os.Remove(filepath)
				}
			}
			if test.bearerTokenFile != "" && test.bearerTokenFileContents != nil {
				filepath := "./" + test.bearerTokenFile
				err := createFile(test.bearerTokenFileContents, filepath)
				require.Nil(t, err)
				defer os.Remove(filepath)
			}

			// Create a HTTP request and add headers to it through an Exporter. Since the
			// Exporter has an empty Headers map, authentication methods will be called.
			exporter := Exporter{
				Config{
					BasicAuth:       test.basicAuth,
					BearerToken:     test.bearerToken,
					BearerTokenFile: test.bearerTokenFile,
				},
			}
			req, err := http.NewRequest(http.MethodPost, server.URL, nil)
			require.Nil(t, err)
			err = exporter.addHeaders(req)

			// Verify the error and if the Authorization header was correctly set.
			if err != nil {
				require.Equal(t, err.Error(), test.expectedError.Error())
			} else {
				require.Nil(t, test.expectedError)
				authHeaderValue := req.Header.Get("Authorization")
				require.Equal(t, authHeaderValue, test.expectedAuthHeaderValue)
			}
		})
	}
}

// createFile writes a file with a slice of bytes at a specified filepath.
func createFile(bytes []byte, filepath string) error {
	err := ioutil.WriteFile(filepath, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

// TestBuildClient checks whether the client returned by buildClient works with TLS and
// has the correct timeout and proxy.
func TestBuildClient(t *testing.T) {
	tests := []struct {
		testName string
	}{}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			// Server
			handler := func(rw http.ResponseWriter, req *http.Request) {
				rw.Write([]byte("test"))
			}
			server := httptest.NewUnstartedServer(http.HandlerFunc(handler))

			// Generate tls config
			generateCACertFiles()
			defer os.Remove("./ca.pem")
			defer os.Remove("./ca_key.pem")

			generateSelfSignedCertFiles()
			defer os.Remove("./cert.pem")
			defer os.Remove("./key.pem")

			tlsCert, err := tls.LoadX509KeyPair("./ca.pem", "./ca_key.pem")
			require.Nil(t, err)

			// Set server TLS and start server
			server.TLS = &tls.Config{
				Certificates: []tls.Certificate{tlsCert},
			}
			server.StartTLS()
			defer server.Close()

			exporter := Exporter{
				Config{
					TLSConfig: map[string]string{
						"ca_file":              "./ca.pem",
						"cert_file":            "./cert.pem",
						"key_file":             "./key.pem",
						"insecure_skip_verify": "1",
					},
				},
			}
			client, err := exporter.buildClient()
			require.Nil(t, err)
			res, err := client.Get(server.URL)
			if err != nil {
				log.Fatalf("could not make GET request: %v", err)
			}
			dump, err := httputil.DumpResponse(res, true)
			if err != nil {
				log.Fatalf("could not dump response: %v", err)
			}
			fmt.Printf("%s\n", dump)

		})
	}
}

func generateCACertFiles() error {
	caCertTemplate := x509.Certificate{
		SerialNumber: big.NewInt(123),
		Subject: pkix.Name{
			Organization: []string{"CA Certificate"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(5 * time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// Generate a key for the new CA certificate.
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create the certificate with the parent certificate as the template.
	caCertBytes, err := x509.CreateCertificate(
		rand.Reader, &caCertTemplate, &caCertTemplate, &privKey.PublicKey, privKey,
	)
	if err != nil {
		return err
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertBytes,
	})
	createFile(caCertPEM, "./ca.pem")

	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	createFile(privKeyPEM, "./ca_key.pem")
	return nil
}

func generateSelfSignedCertFiles() error {
	ssCertTemplate := x509.Certificate{
		SerialNumber: big.NewInt(123),
		Subject: pkix.Name{
			Organization: []string{"CA Certificate"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(5 * time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Generate a key for the new CA certificate.
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	ssCertBytes, err := x509.CreateCertificate(
		rand.Reader, &ssCertTemplate, &ssCertTemplate, &privKey.PublicKey, privKey,
	)
	if err != nil {
		return err
	}

	// Write certificate to cert.pem.
	ssCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ssCertBytes,
	})
	createFile(ssCertPEM, "./cert.pem")

	// Write key to key.pem.
	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	createFile(privKeyPEM, "./key.pem")
	return nil
}
