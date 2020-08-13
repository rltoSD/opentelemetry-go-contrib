package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

func main() {
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	fmt.Println("Got config")

	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(2*time.Second))
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	fmt.Println("Created Pipeline")
	defer pusher.Stop()

	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	recorder := metric.Must(meter).NewInt64ValueRecorder(
		"a.valuerecorder",
		metric.WithDescription("Records values"),
	)

	counter := metric.Must(meter).NewInt64Counter(
		"a.counter",
		metric.WithDescription("Counts things"),
	)

	for i := 1; i <= 10000; i++ {
		time.Sleep(5 * time.Second)
		recorder.Record(ctx, int64(i), kv.String("key", "value"))
		counter.Add(ctx, int64(i), kv.String("key", "value"))
		fmt.Printf("%d. Adding %d to counter and recording %d in recorder\n", i, i, i)
	}

}
