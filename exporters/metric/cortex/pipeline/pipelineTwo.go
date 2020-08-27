package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
	"go.opentelemetry.io/otel/api/metric"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/metrictest"
	"go.opentelemetry.io/otel/sdk/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

func runPipelineTwo() {
	// Read config YAML file to generate a Config struct.
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[Success] Created Config struct")

	// Create an exporter.
	exporter, err := cortex.NewRawExporter(*config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[Success] Created New Cortex Exporter!")

	// Create context.
	ctx := context.Background()

	// Create sum checkpoint set with resource and descriptor
	checkpointSet := metrictest.NewCheckpointSet(nil)
	desc := metric.NewDescriptor("pipeline_two_test", metric.CounterKind, metric.Int64NumberKind)

	// Create aggregation, add value, and update checkpointset
	agg, ckpt := metrictest.Unslice2(sum.New(2))

	// Note: Aggregator tests are written assuming that the SDK
	// has performed the RangeTest. Therefore we skip errors that
	// would have been detected by the RangeTest.
	checkedUpdate(agg, metric.NewInt64Number(123), &desc)
	err = agg.SynchronizedMove(ckpt, &desc)
	if err != nil {
		log.Fatal(err)
	}
	checkpointSet.Add(&desc, ckpt)

	// fmt.Println(time.Time{}.UnixNano() / int64(time.Millisecond))
	// fmt.Println(time.Time{}.Unix())
	// fmt.Println(strconv.Itoa(int(time.Now().Unix())))
	// fmt.Println(strconv.Itoa(int(time.Time{}.Unix())))

	u, err := url.Parse("http://0.0.0.0:9009/api/prom/api/v1/query_range")
	if err != nil {
		log.Println(err)
		return
	}
	q := u.Query()
	q.Add("query", "pipeline_two_test")
	q.Add("start", strconv.Itoa(int(time.Time{}.Unix())))
	q.Add("end", strconv.Itoa(int(time.Now().Unix())))
	q.Add("step", "999999999")
	u.RawQuery = q.Encode()
	fmt.Println("url: ", u)
	// fmt.Println(time.Now().Unix())

	got, err := exporter.ConvertToTimeSeries(checkpointSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(got)

	// Export to Cortex.
	err = exporter.Export(ctx, checkpointSet)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[Success] Exported!")
}

// Performs the same range test the SDK does on behalf of the aggregator.
func checkedUpdate(agg export.Aggregator, number metric.Number, descriptor *metric.Descriptor) {
	ctx := context.Background()

	// Note: Aggregator tests are written assuming that the SDK
	// has performed the RangeTest. Therefore we skip errors that
	// would have been detected by the RangeTest.
	err := aggregator.RangeTest(number, descriptor)
	if err != nil {
		return
	}

	if err := agg.Update(ctx, number, descriptor); err != nil {
		log.Fatal("Unexpected Update failure", err)
	}
}
