<<<<<<< HEAD
=======
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

>>>>>>> upstream-master
package main

import (
	"context"
	"fmt"
<<<<<<< HEAD
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
=======
	"math/rand"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
>>>>>>> upstream-master
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
	"go.opentelemetry.io/otel/sdk/resource"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

func main() {
<<<<<<< HEAD
=======
	// Create a new Config
>>>>>>> upstream-master
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	fmt.Println("Success: Created Config struct")

<<<<<<< HEAD
	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(2*time.Second), push.WithResource(resource.New(kv.String("R", "V"))))
=======
	// Create and install the exporter
	// Optionally, set the push interval to 5 seconds
	// Optionally, add a resource to the controller
	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(5*time.Second), push.WithResource(resource.New(label.String("R", "V"))))
>>>>>>> upstream-master
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	defer pusher.Stop()
	fmt.Println("Success: Installed Exporter Pipeline")

<<<<<<< HEAD
=======
	// Create a counter and a value recorder
>>>>>>> upstream-master
	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	recorder := metric.Must(meter).NewInt64ValueRecorder(
		"pipeline.valuerecorder",
		metric.WithDescription("Records values"),
	)

	counter := metric.Must(meter).NewInt64Counter(
		"pipeline.counter",
		metric.WithDescription("Counts things"),
	)
	fmt.Println("Success: Created Int64ValueRecorder and Int64Counter instruments")

<<<<<<< HEAD
	fmt.Println("Starting to write data to the instruments")
	for i := 1; i <= 10000; i++ {
		time.Sleep(2 * time.Second)
		value := int64(i * 100)
		recorder.Record(ctx, value, kv.String("key", "value"))
		counter.Add(ctx, int64(i), kv.String("key", "value"))
		fmt.Printf("%d. Adding %d to counter and recording %d in recorder\n", i, i, value)
=======
	// Record random values to the instruments in a loop
	fmt.Println("Starting to write data to the instruments")
	seed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(seed)
	for {
		time.Sleep(5 * time.Second)
		randomValue := random.Intn(100)
		value := int64(randomValue * 10)
		recorder.Record(ctx, value, label.String("key", "value"))
		counter.Add(ctx, int64(randomValue), label.String("key", "value"))
		fmt.Printf("Adding %d to counter and recording %d in recorder\n", randomValue, value)
>>>>>>> upstream-master
	}

}
