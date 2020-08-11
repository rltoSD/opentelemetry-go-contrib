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
	"testing"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/api/kv"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregation"
	"go.opentelemetry.io/otel/sdk/resource"
)

var testResource = resource.New(kv.String("R", "V"))
var mockTime int64 = time.Time{}.Unix()

func TestExportKindFor(t *testing.T) {
	exporter := Exporter{}
	got := exporter.ExportKindFor(nil, aggregation.Kind(0))
	want := export.CumulativeExporter

	if got != want {
		t.Errorf("ExportKindFor() =  %q, want %q", got, want)
	}
}

func TestConvertToTimeSeries(t *testing.T) {
	// Setup
	exporter := Exporter{}

	t.Run("handles valid checkpointSet", func(t *testing.T) {

		// Create valid checkpoint set
		validCheckpointSet := getValidCheckpointSet(t)
		// Convert
		got, err := exporter.ConvertToTimeSeries(validCheckpointSet)
		want := []*prompb.TimeSeries{
			&prompb.TimeSeries{
				Labels: []*prompb.Label{
					{
						Name:  "R",
						Value: "V",
					},
					{
						Name:  "name",
						Value: "metric_name",
					},
				},
				Samples: []prompb.Sample{{
					Value:     321,
					Timestamp: mockTime,
				}},
			},
		}

		assert.Nil(t, err, "ConvertToTimeSeries error")
		assert.Len(t, got, 1, "Expected one timeseries")
		assert.ElementsMatch(t, got, want)
	})

	// Test conversions based on aggregation type
	tests := []struct {
		name  string
		input export.CheckpointSet
		want  []*prompb.TimeSeries
	}{
		{
			name:  "convertFromSum",
			input: getSumCheckpoint(t, 321),
			want: []*prompb.TimeSeries{
				&prompb.TimeSeries{
					Labels: []*prompb.Label{
						{
							Name:  "R",
							Value: "V",
						},
						{
							Name:  "name",
							Value: "metric_name",
						},
					},
					Samples: []prompb.Sample{{
						Value:     321,
						Timestamp: mockTime,
					}},
				},
			},
		},
		{
			name:  "convertFromLastValue",
			input: getLastValueCheckpoint(t, 123),
			want: []*prompb.TimeSeries{
				&prompb.TimeSeries{
					Labels: []*prompb.Label{
						{
							Name:  "R",
							Value: "V",
						},
						{
							Name:  "name",
							Value: "metric_name",
						},
					},
					Samples: []prompb.Sample{{
						Value:     123,
						Timestamp: mockTime,
					}},
				},
			},
		},
		// TODO: Add MinMaxSumCount test case
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := exporter.ConvertToTimeSeries(tt.input)
			want := tt.want

			assert.Nil(t, err, "ConvertToTimeSeries error")
			assert.ElementsMatch(t, got, want)
		})
	}
}
