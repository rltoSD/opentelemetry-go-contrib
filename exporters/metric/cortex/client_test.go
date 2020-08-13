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

	"github.com/stretchr/testify/require"
)

// TestSecureTransport checks whether http requests sent using a SecureTransport has the
// correct Authorization header added.
func TestSecureTransport(t *testing.T) {
	tests := []struct {
		testName                string
		expectedAuthHeaderValue string
		expectedError           error
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

			// Create a SecureTransport that adds an Authorization header.
			server.Client().Transport = &SecureTransport{
				rt: http.DefaultTransport,
			}

			// Verify that the Transport successfully added the Authorization header.
			resp, _ := server.Client().Get(server.URL)
			body, _ := ioutil.ReadAll(resp.Body)
			authHeaderValue := string(body)
			require.Equal(t, authHeaderValue, test.expectedAuthHeaderValue)
		})
	}
}
