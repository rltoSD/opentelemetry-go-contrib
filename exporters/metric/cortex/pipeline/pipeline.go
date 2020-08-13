package pipeline

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

func pipeline() error {
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		return err
	}
	// fmt.Println(config)

	pusher, err := cortex.InstallNewPipeline(*config)
	if err != nil {
		return err
	}
	defer pusher.Stop()

	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	recorder := metric.Must(meter).NewInt64ValueRecorder(
		"a.valuerecorder",
		metric.WithDescription("Records values"),
	)

	for i := 1; i <= 10000; i++ {
		time.Sleep(1 * time.Second)
		recorder.Record(ctx, 1, kv.String("key", "value"))
		fmt.Printf("%d. Recording %d in recorder\n", i, 1)
	}

	return nil
}

// type Config struct {
// 	InstrumentConfigs []InstrumentConfig `json:"instrumentConfigs"`
// }

// type InstrumentConfig struct {
// 	Type           string `json:"type"`
// 	Label          string `json:"label"`
// 	Description    string `json:"description"`
// 	DataPointCount int    `json:"dataPointCount"`
// 	RecordInterval int    `json:"recordInterval"`
// 	instrument     interface{}
// }

// func readConfig(filepath string) (*Config, error) {
// 	// Read JSON file.
// 	file, err := ioutil.ReadFile(filepath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var config Config
// 	json.Unmarshal([]byte(file), &config)
// 	return &config, nil
// }

// func prometheusToCortexPipeline() {
// 	// Setup a new Prometheus Exporter
// 	exporter, err := prometheus.NewExportPipeline(
// 		prometheus.Config{},
// 		pull.WithResource(resource.New(kv.String("R", "V"))),
// 	)
// 	if err != nil {
// 		panic(err)
// 	}
// 	meter := exporter.Provider().Meter("example")
// 	ctx := context.Background()

// 	// Read config file.
// 	config, err := readConfig("data_config.json")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	for _, config := range config.InstrumentConfigs {
// 		if config.Type == "COUNTER" {
// 			counter := metric.Must(meter).NewInt64Counter(
// 				config.Label,
// 				metric.WithDescription(config.Description),
// 			)
// 			config.instrument = counter

// 			go func(counter metric.Int64Counter) {
// 				for i := 1; i <= config.DataPointCount; i++ {
// 					time.Sleep(time.Duration(config.RecordInterval) * time.Second)
// 					counter.Add(ctx, 1, kv.String("key", "value"))
// 					fmt.Printf("%d. Adding 1 to counter\n", i)
// 				}
// 			}(counter)
// 		}
// 		if config.Type == "VALUERECORDER" {
// 			recorder := metric.Must(meter).NewInt64ValueRecorder(
// 				config.Label,
// 				metric.WithDescription(config.Description),
// 			)
// 			config.instrument = recorder

// 			go func(recorder metric.Int64ValueRecorder) {
// 				for i := 1; i <= config.DataPointCount; i++ {
// 					time.Sleep(time.Duration(config.RecordInterval) * time.Second)
// 					recorder.Record(ctx, 1, kv.String("key", "value"))
// 					fmt.Printf("%d. Recording %d in recorder\n", i, 1)
// 				}
// 			}(recorder)
// 		}
// 	}

// 	// Set up an endpoint to wait for Prometheus scrapes
// 	fmt.Println("Server started!")
// 	http.Handle("/", exporter)
// 	http.ListenAndServe(":8888", nil)
// }
