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
	"encoding/base64"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestSecureTransport checks whether http requests sent using a SecureTransport has the
// correct Authorization header added.
func TestSecureTransport(t *testing.T) {
	tests := []struct {
		testName                string
		basicAuth               map[string]string
		bearerToken             string
		bearerTokenFile         string
		expectedAuthHeaderValue string
		expectedError           error
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

			// Create a SecureTransport that adds an Authorization header.
			server.Client().Transport = &SecureTransport{
				basicAuth:       test.basicAuth,
				bearerToken:     test.bearerToken,
				bearerTokenFile: test.bearerTokenFile,
				rt:              http.DefaultTransport,
			}

			// Verify that the Transport successfully added the Authorization header.
			resp, err := server.Client().Get(server.URL)
			if err != nil {
				// Error will be of form: GET "<server url>": error. The server URL
				// changes each time a test is run, so the test only checks the last part
				// of the error string.
				fullError := err.Error()
				splitError := strings.Split(fullError, ": ")
				errorString := splitError[len(splitError)-1]
				require.Equal(t, errorString, test.expectedError.Error())
			} else {
				require.Nil(t, test.expectedError)

				// Read body and verify the header value.
				body, err := ioutil.ReadAll(resp.Body)
				require.Nil(t, err)
				authHeaderValue := string(body)
				require.Equal(t, authHeaderValue, test.expectedAuthHeaderValue)
			}
		})
	}
}
