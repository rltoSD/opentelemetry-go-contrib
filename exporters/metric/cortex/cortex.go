package cortex

import (
	"context"

	apimetric "go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/metric/selector/simple"
)

// Exporter forwards metrics to a Cortex
type Exporter struct {
	Config
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
