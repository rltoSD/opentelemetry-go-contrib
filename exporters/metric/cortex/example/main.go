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

// This is an example program that creates metrics
// and exports to Cortex.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/global"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
)

// Creates a Cortex Exporter
func initMeter() {
	exporter, err := cortex.InstallNewPipeline(cortex.Config{})
	if err != nil {
		log.Panicf("Failed to initialize Cortex exporter %v", err)
	}
	fmt.Println("Cortex exporter now running")
}

func main() {
	// Create Exporter
	initMeter()

	// Get global meter, labels, and context
	meter := global.Meter("ex.com/basic")
	commonLabels := []kv.KeyValue{lemonsKey.Int(10), kv.String("A", "1"), kv.String("B", "2"), kv.String("C", "3")}
	ctx := context.Background()

	// Create a counter
	counter := metric.Must(meter).NewFloat64Counter("float64_counter")

	// While the program is running, increment the counter
	for x := range time.Tick(1 * time.Second) {
		fmt.Printf("Tick: %v\n", x)
		counter.add(ctx, 1, commonLabels)
	}
}
