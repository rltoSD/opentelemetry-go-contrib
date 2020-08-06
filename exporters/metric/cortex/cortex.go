package cortex

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"go.opentelemetry.io/otel/api/global"
	apimetric "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

// Exporter forwards metrics to a Cortex
type Exporter struct {
	Config Config
}

// ExportKindFor returns CumulativeExporter so the Processor correctly aggregates data
func (e *Exporter) ExportKindFor(*apimetric.Descriptor, aggregation.Kind) metric.ExportKind {
	return metric.CumulativeExporter
}

// Export forwards metrics to Cortex from the SDK
func (e *Exporter) Export(_ context.Context, checkpointSet metric.CheckpointSet) error {
	return nil
}

// NewRawExporter validates the Config struct and creates an Exporter with it.
func NewRawExporter(config Config) (*Exporter, error) {
	// This is redundant when the user creates the Config struct with the NewConfig function.
	if err := config.Validate(); err != nil {
		return nil, err
	}

	exporter := Exporter{config}
	return &exporter, nil
}

// NewExportPipeline sets up a complete export pipeline with a push Controller and Exporter.
func NewExportPipeline(config Config, options ...push.Option) (*push.Controller, error) {
	exporter, err := NewRawExporter(config)
	if err != nil {
		return nil, err
	}

	pusher := push.New(
		simple.NewWithExactDistribution(),
		exporter,
		options...,
	)
	pusher.Start()
	return pusher, nil
}

// InstallNewPipeline registers a push Controller's Provider globally.
func InstallNewPipeline(config Config, options ...push.Option) (*push.Controller, error) {
	pusher, err := NewExportPipeline(config, options...)
	if err != nil {
		return nil, err
	}
	global.SetMeterProvider(pusher.Provider())
	return pusher, nil
}

// AddHeaders adds required headers as well as all headers in Header map to a http request.
func (e *Exporter) AddHeaders(req *http.Request) {
	// Cortex expects Snappy-compressed protobuf messages. These two headers are hard-coded as they
	// should be on every request.
	req.Header.Add("Content-Encoding", "snappy")
	req.Header.Set("Content-Type", "application/x-protobuf")

	// Add all user-supplied headers to the request.
	for name, field := range e.Config.Headers {
		req.Header.Add(name, field)
	}
}

// BuildRequest creates an http POST request with a []byte as the body and headers attached.
func (e *Exporter) BuildRequest(message []byte) (*http.Request, error) {
	// Create the request with the endpoint and message. The message should be a Snappy-compressed
	// protobuf message.
	req, err := http.NewRequest("POST", e.Config.Endpoint, bytes.NewBuffer(message))
	if err != nil {
		return nil, err
	}

	// Add the required headers and the headers from Config.Headers.
	e.AddHeaders(req)

	return req, nil
}

// BuildMessage creates a Snappy-compressed protobuf message from a slice of TimeSeries.
func (e *Exporter) BuildMessage(timeseries []*prompb.TimeSeries) ([]byte, error) {
	// Wrap the TimeSeries as a WriteRequest since Cortex requires it.
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeseries,
	}

	// Convert the struct to a slice of bytes.
	message, err := proto.Marshal(writeRequest)
	if err != nil {
		return nil, err
	}

	// Compress the message.
	compressed := snappy.Encode(message, nil)

	return compressed, nil
}

// Default number of times to retry sending a request upon failure.
var maxRetryCount = 1

// ErrRetryLimitReached is an error for when the Exporter fails to send the request after more than
// `maxRetryCount` times.
var ErrRetryLimitReached = fmt.Errorf("Failed to send request after reaching retry limit")

// SendRequest sends an http request using the Exporter's http Client. It will retry once
func (e *Exporter) SendRequest(req *http.Request, retryCount int) (int, error) {
	// Attempt to send request.
	res, err := e.Config.Client.Do(req)
	if err != nil {
		return -1, err
	}

	// Request was successfully sent if the request status code is 2xx.
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return res.StatusCode, nil
	}

	// Retry up to `maxRetryCount`. Otherwise, return an error.
	retryCount++
	if retryCount > maxRetryCount {
		return res.StatusCode, ErrRetryLimitReached
	}
	return e.SendRequest(req, retryCount)
}
