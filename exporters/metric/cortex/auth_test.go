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
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

// createFile writes a file with a slice of bytes at a specified filepath.
func createFile(bytes []byte, filepath string) error {
	err := ioutil.WriteFile(filepath, bytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

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
	}{}
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
		})
	}
}
