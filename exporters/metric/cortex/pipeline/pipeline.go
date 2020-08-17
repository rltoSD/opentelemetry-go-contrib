package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

func main() {
	// Creates a push controller that calls Export() every 2 seconds.
	pusher, err := initPipeline(2 * time.Second)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer pusher.Stop()

	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	int64Counter, _, _, _, int64ValueRecorder, _, _, _, _, _, _, _ := initInstruments(meter)
	fmt.Println("Success: Created all instruments")

	// int64Counter, float64Counter, int64UpDownCounter, float64UpDownCounter, int64ValueRecorder, float64ValueRecorder, int64ValueObserver, float64ValueObserver, int64SumObserver, float64SumObserver, int64UpDownSumObserver, float64UpDownSumObserver := initInstruments(meter)
	// fmt.Println("Success: Created all instruments")

	fmt.Println("Starting to write data to the instruments")
	for i := 1; i <= 10000; i++ {
		time.Sleep(2 * time.Second)
		value := int64(i * 100)
		int64ValueRecorder.Record(ctx, value, kv.String("key", "value"))
		int64Counter.Add(ctx, int64(i), kv.String("key", "value"))
		fmt.Printf("%d. Adding %d to counter and recording %d in recorder\n", i, i, value)
	}

}

// initPipeline runs the Exporter setup pipeline to create a new Exporter and push
// Controller.
func initPipeline(pushInterval time.Duration) (*push.Controller, error) {
	// Read config YAML file to generate a Config struct.
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		return nil, err
	}
	fmt.Println("Success: Created Config struct")

	// Run exporter setup pipeline.
	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(pushInterval))
	if err != nil {
		return nil, err
	}
	fmt.Println("Success: Installed Exporter Pipeline")

	return pusher, nil
}

func initInstruments(meter metric.Meter) (metric.Int64Counter, metric.Float64Counter, metric.Int64UpDownCounter, metric.Float64UpDownCounter, metric.Int64ValueRecorder, metric.Float64ValueRecorder, metric.Int64ValueObserver, metric.Float64ValueObserver, metric.Int64SumObserver, metric.Float64SumObserver, metric.Int64UpDownSumObserver, metric.Float64UpDownSumObserver) {
	int64Counter := metric.Must(meter).NewInt64Counter(
		"pipeline.int64Counter",
		metric.WithDescription("Synchronous additive monotonic counter for 64-bit integers"),
	)

	float64Counter := metric.Must(meter).NewFloat64Counter(
		"pipeline.float64Counter",
		metric.WithDescription("Synchronous additive monotonic counter for 64-bit floats"),
	)

	int64UpDownCounter := metric.Must(meter).NewInt64UpDownCounter(
		"pipeline.int64UpDownCounter",
		metric.WithDescription("Synchronous additive instrument for 64-bit integers"),
	)

	float64UpDownCounter := metric.Must(meter).NewFloat64UpDownCounter(
		"pipeline.float64UpDownCounter",
		metric.WithDescription("Synchronous additive instrument for 64-bit floats"),
	)

	int64ValueRecorder := metric.Must(meter).NewInt64ValueRecorder(
		"pipeline.int64ValueRecorder",
		metric.WithDescription("Non-additive synchronous instrument for 64-bit integers"),
	)

	float64ValueRecorder := metric.Must(meter).NewFloat64ValueRecorder(
		"pipeline.float64ValueRecorder",
		metric.WithDescription("Non-additive synchronous instrument for 64-bit floats"),
	)

	int64ValueObserver := metric.Must(meter).NewInt64ValueObserver(
		"pipeline.int64ValueObserver",
		func(context.Context, metric.Int64ObserverResult) {},
		metric.WithDescription("Non-additive asynchronous instrument for 64-bit integers"),
	)

	float64ValueObserver := metric.Must(meter).NewFloat64ValueObserver(
		"pipeline.float64ValueObserver",
		func(context.Context, metric.Float64ObserverResult) {},
		metric.WithDescription("Non-additive asynchronous instrument for 64-bit floats"),
	)

	int64SumObserver := metric.Must(meter).NewInt64SumObserver(
		"pipeline.int64SumObserver",
		func(context.Context, metric.Int64ObserverResult) {},
		metric.WithDescription("Asynchronous additive monotonic instrument for 64-bit integers"),
	)

	float64SumObserver := metric.Must(meter).NewFloat64SumObserver(
		"pipeline.float64SumObserver",
		func(context.Context, metric.Float64ObserverResult) {},
		metric.WithDescription("Asynchronous additive monotonic instrument for 64-bit integers"),
	)

	int64UpDownSumObserver := metric.Must(meter).NewInt64UpDownSumObserver(
		"pipeline.int64UpDownSumObserver",
		func(context.Context, metric.Int64ObserverResult) {},
		metric.WithDescription("Asynchronous additive instrument for 64-bit integers"),
	)

	float64UpDownSumObserver := metric.Must(meter).NewFloat64UpDownSumObserver(
		"pipeline.float64UpDownSumObserver",
		func(context.Context, metric.Float64ObserverResult) {},
		metric.WithDescription("Asynchronous additive monotonic instrument for 64-bit integers"),
	)

	return int64Counter, float64Counter, int64UpDownCounter, float64UpDownCounter, int64ValueRecorder, float64ValueRecorder, int64ValueObserver, float64ValueObserver, int64SumObserver, float64SumObserver, int64UpDownSumObserver, float64UpDownSumObserver
}
