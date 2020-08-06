package cortex_test

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

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
