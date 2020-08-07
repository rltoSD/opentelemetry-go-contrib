package cortex_test

import (
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
	"opentelemetry.io/contrib/exporters/metric/cortex"
)

func TestExportKindFor(t *testing.T) {
	exporter := cortex.Exporter{}
	got := exporter.ExportKindFor(nil, aggregation.Kind(0))
	want := metric.CumulativeExporter

	if got != want {
		t.Errorf("ExportKindFor() =  %q, want %q", got, want)
	}
}

// TestNewRawExporter tests whether NewRawExporter successfully creates an Exporter with the same
// Config struct as the one passed in.
func TestNewRawExporter(t *testing.T) {
	exporter, err := cortex.NewRawExporter(ValidConfig)
	if err != nil {
		t.Fatalf("Failed to create exporter with error %v", err)
	}

	if !cmp.Equal(ValidConfig, exporter.Config) {
		t.Fatalf("Got configuration %v, wanted %v", exporter.Config, ValidConfig)
	}
}

// TestNewExportPipeline tests whether a push Controller was successfully created with an Exporter
// from New RawExporter. Errors in this function will be from calls to push controller package and
// NewRawExport. Both have their own tests.
func TestNewExportPipeline(t *testing.T) {
	_, err := cortex.NewExportPipeline(ValidConfig)
	if err != nil {
		t.Fatalf("Failed to create export pipeline with error %v", err)
	}
}

// TestInstallNewPipeline checks whether InstallNewPipeline successfully returns a push Controller
// and whether that controller's Provider is registered globally.
func TestInstallNewPipeline(t *testing.T) {
	pusher, err := cortex.InstallNewPipeline(ValidConfig)
	if err != nil {
		t.Fatalf("Failed to create install pipeline with error %v", err)
	}
	if global.MeterProvider() != pusher.Provider() {
		t.Fatalf("Failed to register push Controller provider globally")
	}
}

// TestAddHeaders tests whether the correct headers are correctly added to an http request.
// Note: this could be moved to a `cortex_internal_test.go` file as it doesn't need to be exported.
func TestAddHeaders(t *testing.T) {
	// Make a fake Config struct and Exporter for testing.
	testConfig := cortex.Config{
		Headers: map[string]string{
			"testHeader":    "testField",
			"TestHeaderTwo": "testFieldTwo",
		},
	}
	exporter := cortex.Exporter{testConfig}

	// Create http request to add headers to.
	req, err := http.NewRequest("POST", "test.com", nil)
	if err != nil {
		t.Errorf("Failed to create http request with error %v", err)
	}
	exporter.AddHeaders(req)

	// Check that all the headers are there.
	for name, field := range testConfig.Headers {
		// Headers are case-insensitive; Viper converts all keys to lower-case.
		lowercaseName := strings.ToLower(name)
		if req.Header.Get(lowercaseName) != field {
			t.Errorf("Failed to add header: '%v' from Config.Headers", name)
		}
	}
	if req.Header.Get("Content-Encoding") != "snappy" {
		t.Errorf("Failed to add required header 'Content-Encoding'")
	}
	if req.Header.Get("Content-Type") != "application/x-protobuf" {
		t.Errorf("Failed to add required header 'Content-Encoding'")
	}
}

// TestBuildRequest tests whether a http request is a POST request, has the correct body, and has
// the correct headers.
// Note: this could be moved to a `cortex_internal_test.go` file as it doesn't need to be exported.
func TestBuildRequest(t *testing.T) {
	// Make fake exporter and message for testing.
	var testMessage = []byte(`Test Message!`)
	exporter := cortex.Exporter{ValidConfig}

	// Create the http request.
	req, err := exporter.BuildRequest(testMessage)
	if err != nil {
		t.Fatalf("Failed to build request with error %v", err)
	}

	// Verify the http method, url, and body.
	if req.Method != http.MethodPost {
		t.Errorf("Request is of method %v, wanted POST", req.Method)
	}
	if req.URL.String() != ValidConfig.Endpoint {
		t.Errorf("Request has endpoint %v, wanted %v", req.URL, ValidConfig.Endpoint)
	}
	reqMessage, err := ioutil.ReadAll(req.Body)
	if err != nil {
		t.Errorf("Failed to read request body with error %v", err)
	}
	if !cmp.Equal(reqMessage, testMessage) {
		t.Errorf("Request body has message %v, wanted %v", reqMessage, testMessage)
	}

	// Verify headers.
	for name, field := range exporter.Config.Headers {
		// Headers are case-insensitive; Viper converts all keys to lower-case.
		lowercaseName := strings.ToLower(name)
		if req.Header.Get(lowercaseName) != field {
			t.Errorf("Failed to add header: '%v' from Config.Headers", name)
		}
	}
	if req.Header.Get("Content-Encoding") != "snappy" {
		t.Errorf("Failed to add required header 'Content-Encoding'")
	}
	if req.Header.Get("Content-Type") != "application/x-protobuf" {
		t.Errorf("Failed to add required header 'Content-Encoding'")
	}
}

// TestBuildMessage tests whether BuildMessage successfully returns a Snappy-compressed protobuf
// message.
// Note: Not too sure how to test this function.
func TestBuildMessage(t *testing.T) {
	exporter := cortex.Exporter{ValidConfig}
	timeseries := []*prompb.TimeSeries{}

	// BuildMessage simply calls protobuf.Marshal() and snappy.Encode(). BuildMessage returns the
	// error returned by these two functions, which have their own tests in their respective
	// packages. As long as no error is returned, the function should work as expected.
	_, err := exporter.BuildMessage(timeseries)
	if err != nil {
		t.Errorf("Failed to build Snappy-compressed protobuf message with error %v", err)
	}
}

// TestSendRequest tests if the Exporter can successfully send a http request as well as the retry
// functionality by creating a test server and sending requests to it using SendRequest(). The test
// server will imitate a failure by returning status code 404 a test-specified amount of times.
// Note: this could be moved to a `cortex_internal_test.go` file as it doesn't need to be exported.
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
			cortex.ErrSendRequestFailure,
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
			customConfig := ValidConfig
			customConfig.Endpoint = server.URL
			customConfig.Headers = map[string]string{
				"isStatusNotFound": strconv.FormatBool(test.isStatusNotFound),
			}
			exporter := cortex.Exporter{customConfig}

			// Create an empty Snappy-compressed message.
			msg, err := exporter.BuildMessage([]*prompb.TimeSeries{})
			require.Nil(t, err)

			// Create a http POST request with the compressed message.
			req, err := exporter.BuildRequest(msg)
			require.Nil(t, err)

			// Send the request to the test server and verify errors and status codes.
			statusCode, err := exporter.SendRequest(req)
			require.Equal(t, err, test.expectedError)
			require.Equal(t, statusCode, test.expectedStatusCode)
		})
	}
}
