package cortex

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/google/go-cmp/cmp"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
)

// ValidConfig is a Config struct that should cause no errors.
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
	ProxyURL:     "",
	PushInterval: "5s",
	Headers: map[string]string{
		"x-prometheus-remote-write-version": "0.1.0",
		"tenant-id":                         "123",
	},
	Client: http.DefaultClient,
}

func TestExportKindFor(t *testing.T) {
	exporter := Exporter{}
	got := exporter.ExportKindFor(nil, aggregation.Kind(0))
	want := metric.CumulativeExporter

	if got != want {
		t.Errorf("ExportKindFor() =  %q, want %q", got, want)
	}
}

// TestNewRawExporter tests whether NewRawExporter successfully creates an Exporter with the same
// Config struct as the one passed in.
func TestNewRawExporter(t *testing.T) {
	exporter, err := NewRawExporter(validConfig)
	if err != nil {
		t.Fatalf("Failed to create exporter with error %v", err)
	}

	if !cmp.Equal(validConfig, exporter.config) {
		t.Fatalf("Got configuration %v, wanted %v", exporter.config, validConfig)
	}
}

// TestNewExportPipeline tests whether a push Controller was successfully created with an Exporter
// from New RawExporter. Errors in this function will be from calls to push controller package and
// NewRawExport. Both have their own tests.
func TestNewExportPipeline(t *testing.T) {
	_, err := NewExportPipeline(validConfig)
	if err != nil {
		t.Fatalf("Failed to create export pipeline with error %v", err)
	}
}

// TestInstallNewPipeline checks whether InstallNewPipeline successfully returns a push Controller
// and whether that controller's Provider is registered globally.
func TestInstallNewPipeline(t *testing.T) {
	pusher, err := InstallNewPipeline(validConfig)
	if err != nil {
		t.Fatalf("Failed to create install pipeline with error %v", err)
	}
	if global.MeterProvider() != pusher.Provider() {
		t.Fatalf("Failed to register push Controller provider globally")
	}
}

// TestAddHeaders tests whether the correct headers are correctly added to an http request.
func TestAddHeaders(t *testing.T) {
	// Make a fake Config struct and Exporter for testing.
	testConfig := Config{
		Headers: map[string]string{
			"testHeader":    "testField",
			"TestHeaderTwo": "testFieldTwo",
		},
	}
	exporter := Exporter{testConfig}

	// Create http request to add headers to.
	req, err := http.NewRequest("POST", "test.com", nil)
	require.Nil(t, err)
	exporter.addHeaders(req)

	// Check that all the headers are there.
	for name, field := range testConfig.Headers {
		// Headers are case-insensitive; Viper converts all keys to lower-case.
		lowercaseName := strings.ToLower(name)
		require.Equal(t, req.Header.Get(lowercaseName), field)
	}
	require.Equal(t, req.Header.Get("Content-Encoding"), "snappy")
	require.Equal(t, req.Header.Get("Content-Type"), "application/x-protobuf")
}

// TestBuildRequest tests whether a http request is a POST request, has the correct body, and has
// the correct headers.
func TestBuildRequest(t *testing.T) {
	// Make fake exporter and message for testing.
	var testMessage = []byte(`Test Message!`)
	exporter := Exporter{validConfig}

	// Create the http request.
	req, err := exporter.buildRequest(testMessage)
	require.Nil(t, err)

	// Verify the http method, url, and body.
	require.Equal(t, req.Method, http.MethodPost)
	require.Equal(t, req.URL.String(), validConfig.Endpoint)

	reqMessage, err := ioutil.ReadAll(req.Body)
	require.Nil(t, err)
	require.Equal(t, reqMessage, testMessage)

	// Verify headers.
	for name, field := range exporter.config.Headers {
		// Headers are case-insensitive; Viper converts all keys to lower-case.
		lowercaseName := strings.ToLower(name)
		require.Equal(t, req.Header.Get(lowercaseName), field)
	}
	require.Equal(t, req.Header.Get("Content-Encoding"), "snappy")
	require.Equal(t, req.Header.Get("Content-Type"), "application/x-protobuf")
}

// TestBuildMessage tests whether BuildMessage successfully returns a Snappy-compressed protobuf
// message.
func TestBuildMessage(t *testing.T) {
	exporter := Exporter{validConfig}
	timeseries := []*prompb.TimeSeries{}

	// BuildMessage simply calls protobuf.Marshal() and snappy.Encode(). BuildMessage returns the
	// error returned by these two functions, which have their own tests in their respective
	// packages. As long as no error is returned, the function should work as expected.
	_, err := exporter.buildMessage(timeseries)
	require.Nil(t, err)
}

// TestSendRequest tests if the Exporter can successfully send a http request with a correctly
// formatted request and the correct headers. A test server returns status codes to test if the
// Exporter responds to send failure correctly.
func TestSendRequest(t *testing.T) {
	tests := []struct {
		name               string
		expectedStatusCode int
		expectedError      error
		isStatusNotFound   bool
	}{
		{
			"Successful Export",
			200,
			nil,
			false,
		},
		{
			"Export Failure",
			404,
			fmt.Errorf("Failed to send the HTTP request with status code %v", 404),
			true,
		},
	}

	// Set up a test server to receive data.
	handler := func(rw http.ResponseWriter, req *http.Request) {
		// Check the request body and make sure it was formatted correctly and has the correct
		// headers.
		compressed, err := ioutil.ReadAll(req.Body)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		uncompressed, err := snappy.Decode(nil, compressed)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		wr := &prompb.WriteRequest{}
		err = proto.Unmarshal(uncompressed, wr)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}
		if req.Header.Get("X-Prometheus-Remote-Write-Version") != "0.1.0" ||
			req.Header.Get("Content-Encoding") != "snappy" ||
			req.Header.Get("Content-Type") != "application/x-protobuf" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return a status code 400 if header isStatusNotFound is "true", 200 otherwise.
		if req.Header.Get("isStatusNotFound") == "true" {
			rw.WriteHeader(http.StatusNotFound)
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set up an Exporter that uses the test server's endpoint and attaches the test's
			// serverFailure header.
			customConfig := validConfig
			customConfig.Endpoint = server.URL
			customConfig.Headers = map[string]string{
				"isStatusNotFound": strconv.FormatBool(test.isStatusNotFound),
			}
			exporter := Exporter{customConfig}

			// Create an empty Snappy-compressed message.
			msg, err := exporter.buildMessage([]*prompb.TimeSeries{})
			require.Nil(t, err)

			// Create a http POST request with the compressed message.
			req, err := exporter.buildRequest(msg)
			require.Nil(t, err)

			// Send the request to the test server and verify errors and status codes.
			err = exporter.sendRequest(req)
			var statusCode int
			var errString string
			if err != nil {
				errString = err.Error()

				// Retrieve status code from error string.
				split := strings.Split(errString, " ")
				statusCode, err = strconv.Atoi(split[len(split)-1])
				require.Nil(t, err)

				// Verify errors.
				require.Equal(t, errString, test.expectedError.Error())
			} else {
				statusCode = 200
				require.Equal(t, nil, test.expectedError)
			}
			require.Equal(t, statusCode, test.expectedStatusCode)
		})
	}
}
