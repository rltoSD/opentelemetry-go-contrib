package cortex_test

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/prometheus/prompb"

	"github.com/google/go-cmp/cmp"
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
	for name, field := range exporter.Headers {
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
		numFailures        int
	}{
		{
			"Successful Export",
			200,
			nil,
			0,
		},
		{
			"Fails Export once",
			200,
			nil,
			1,
		},
		{
			"Fail Export twice",
			404,
			cortex.ErrRetryLimitReached,
			2,
		},
	}

	// This value will be set in the testing loop and is used by the handler function.
	failureCount := 0

	// Set up a test server to receive data. The handler function will return status code
	// 404 and decrement this value until it reaches 0, when it returns status code 200.
	handler := func(rw http.ResponseWriter, req *http.Request) {
		if failureCount > 0 {
			failureCount--
			rw.WriteHeader(http.StatusNotFound)
		} else {
			rw.WriteHeader(http.StatusOK)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Set number of times the server will return 404 on.
			failureCount = test.numFailures

			// Set up an Exporter that uses the test server's endpoint.
			customConfig := ValidConfig
			customConfig.Endpoint = server.URL
			exporter := cortex.Exporter{customConfig}

			// Create an empty Snappy-compressed message.
			msg, err := exporter.BuildMessage([]*prompb.TimeSeries{})
			if err != nil {
				t.Fatalf("Failed to build Snappy-compressed protobuf message with error %v", err)
			}

			// Create a http POST request with the compressed message.
			req, err := exporter.BuildRequest(msg)
			if err != nil {
				t.Fatalf("Failed to build request with error %v", err)
			}

			// Send the request to the test server and verify errors and status codes.
			statusCode, err := exporter.SendRequest(req, 0)
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("Wanted error %v, received error %v", test.expectedError, err)
			}
			if statusCode != test.expectedStatusCode {
				t.Errorf("Wanted status code %v, received status code %v instead", test.expectedStatusCode, statusCode)
			}
		})
	}
}
