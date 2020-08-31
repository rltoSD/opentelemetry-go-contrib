package main

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
)

var labels = []label.KeyValue{
	label.String("key1", "value1"),
	label.String("key2", "value2"),
	label.String("key3", "value3"),
	label.String("key4", "value4"),
	label.String("key5", "value5"),
}
var values = createValues(100000)
var checkpointSet = buildCheckpointSet("dist", "benchmark_dist", labels, values, metric.ValueRecorderKind)

var exporter = initPipelineTwo()
var ctx = context.Background()

func createValues(numValues int) []int64 {
	var values []int64
	for i := 0; i < numValues; i++ {
		val := int64(i)
		values = append(values, val)
	}
	return values
}

func BenchmarkExport(b *testing.B) {
	var err error
	for n := 0; n < b.N; n++ {
		err = exporter.Export(ctx, checkpointSet)
	}
	if err != nil {
		b.Errorf("Error: %v", err)
	}
}
