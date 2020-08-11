package cortex

import (
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/metrictest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/aggregatortest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

// getLabels returns labels from pairs of strings
func getLabels(labels ...string) []prompb.Label {
	pbLabels := prompb.Labels{
		Labels: []prompb.Label{},
	}
	for i := 0; i < len(labels); i += 2 {
		pbLabels.Labels = append(pbLabels.Labels, *getLabel(labels[i], labels[i+1]))
	}
	return pbLabels.Labels
}

// getLabel returns a Label given a name and value
func getLabel(name string, value string) *prompb.Label {
	return &prompb.Label{
		Name:  name,
		Value: value,
	}
}

// getSample returns a sample given a value and timestamp
func getSample(value float64, timestamp int64) prompb.Sample {
	return prompb.Sample{
		Value:     value,
		Timestamp: timestamp,
	}
}

// getTimeSeries returns a timeseries containing labels and samples
func getTimeSeries(labels []*prompb.Label, samples ...prompb.Sample) *prompb.TimeSeries {
	return &prompb.TimeSeries{
		Labels:  labels,
		Samples: samples,
	}
}

// getValidCheckpointSet returns a valid checkpointset with several records
func getValidCheckpointSet(t *testing.T) export.CheckpointSet {
	return getSumCheckpoint(t, 321)
}

// getSumCheckpoint returns a checkpoint set with a sum aggregation record
func getSumCheckpoint(t *testing.T, value int64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.CounterKind, metric.Int64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(sum.New(2))
	aggregatortest.CheckedUpdate(t, agg, metric.NewInt64Number(value), &desc)
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}

// getLastValueCheckpoint returns a checkpoint set with a last value aggregation record
func getLastValueCheckpoint(t *testing.T, value int64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.ValueObserverKind, metric.Int64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(lastvalue.New(2))
	aggregatortest.CheckedUpdate(t, agg, metric.NewInt64Number(value), &desc)
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}

// getMMSCCheckpoint returns a checkpoint set with a minmaxsumcount aggregation record
func getMMSCCheckpoint(t *testing.T, values ...float64) export.CheckpointSet {
	// Create checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(testResource)
	desc := metric.NewDescriptor("metric_name", metric.ValueRecorderKind, metric.Float64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(minmaxsumcount.New(2, &desc))
	for _, value := range values {
		aggregatortest.CheckedUpdate(t, agg, metric.NewFloat64Number(value), &desc)
	}
	require.NoError(t, agg.SynchronizedMove(ckpt, &desc))
	checkpointSet.Add(&desc, ckpt)

	return checkpointSet
}
