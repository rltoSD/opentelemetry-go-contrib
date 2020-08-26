package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/exporters/metric/cortex"
	"go.opentelemetry.io/otel/api/kv"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"

	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
)

// Pipeline settings.
// var pipelineOneFilename string = "data/test.csv"

var pipelineOneFilename string = "data/PrometheusDataFirst.csv"

// var pipelineOneSleepPeriod time.Duration = 1 * time.Second

var pipelineOneSleepPeriod time.Duration = 50 * time.Millisecond

func main() {
	// Start a timer to measure how long pipeline test takes.
	start := time.Now()
	fmt.Printf("Starting pipeline test!\n\n")

	runPipelineOne()

	// Print out elapsed time.
	elapsed := time.Since(start)
	fmt.Printf("\n[Success] Completed pipeline test!\n")
	fmt.Printf("Elapsed Time: %v\n", elapsed)
}

// runPipelineOne runs a pipeline that records values to various instruments and exports
// metrics data to Cortex.
func runPipelineOne() {
	// Creates a push controller that calls Export() every 2 seconds.
	pusher, err := initPipeline(100 * time.Millisecond)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer pusher.Stop()

	// Create a csv reader for reading the input data.
	reader, err := initCSVReader(pipelineOneFilename)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create the meter for creating synchronous instruments.
	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	// Iterate through the CSV file line by line and record data to the instruments.
	for i := 1; i > 0; i++ {
		// Retrieve the next line from the CSV file.
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// Parse the next record.
		instrument, valueStr, name, desc, keyValuePairs, err := parsePipelineOneRecord(record)
		if err != nil {
			log.Fatal(err)
		}

		// All values in the generated data are integers, but are read as strings.
		val, err := strconv.Atoi(valueStr)
		if err != nil {
			log.Fatalf("Failed to parse %v as an integer", valueStr)
		}

		// Record the data in the correct instrument.
		invalidRecord := false
		switch instrument {
		case "ictr":
			i := metric.Must(meter).NewInt64Counter(name, metric.WithDescription(desc))
			i.Add(ctx, int64(val), keyValuePairs...)
		case "fctr":
			i := metric.Must(meter).NewFloat64Counter(name, metric.WithDescription(desc))
			i.Add(ctx, float64(val), keyValuePairs...)
		case "iudctr":
			i := metric.Must(meter).NewInt64UpDownCounter(name, metric.WithDescription(desc))
			i.Add(ctx, int64(val), keyValuePairs...)
		case "fudctr":
			i := metric.Must(meter).NewFloat64UpDownCounter(name, metric.WithDescription(desc))
			i.Add(ctx, float64(val), keyValuePairs...)
		case "ivrec":
			i := metric.Must(meter).NewInt64ValueRecorder(name, metric.WithDescription(desc))
			i.Record(ctx, int64(val), keyValuePairs...)
		case "fvrec":
			i := metric.Must(meter).NewFloat64ValueRecorder(name, metric.WithDescription(desc))
			i.Record(ctx, float64(val), keyValuePairs...)
		case "isobs":
			newInt64SumObserverRecord = true
			_ = metric.Must(meter).NewInt64SumObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					if newInt64SumObserverRecord {
						result.Observe(int64(val), keyValuePairs...)
						newInt64SumObserverRecord = false
					}
				},
				metric.WithDescription(desc),
			)
		case "fsobs":
			newFloat64SumObserverRecord = true
			_ = metric.Must(meter).NewFloat64SumObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					if newFloat64SumObserverRecord {
						result.Observe(float64(val), keyValuePairs...)
						newFloat64SumObserverRecord = false
					}
				},
				metric.WithDescription(desc),
			)
		case "iudobs":
			newInt64UpDownSumObserverRecord = true
			_ = metric.Must(meter).NewInt64UpDownSumObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					if newInt64UpDownSumObserverRecord {
						result.Observe(int64(val), keyValuePairs...)
						newInt64UpDownSumObserverRecord = false
					}
				},
				metric.WithDescription(desc),
			)
		case "fudobs":
			newFloat64UpDownSumObserverRecord = true
			_ = metric.Must(meter).NewFloat64UpDownSumObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					if newFloat64UpDownSumObserverRecord {
						result.Observe(float64(val), keyValuePairs...)
						newFloat64UpDownSumObserverRecord = false
					}
				},
				metric.WithDescription(desc),
			)
		case "ivobs":
			newInt64ValueObserverRecord = true
			_ = metric.Must(meter).NewInt64ValueObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					if newInt64ValueObserverRecord {
						result.Observe(int64(val), keyValuePairs...)
						newInt64ValueObserverRecord = false
					}
				},
				metric.WithDescription(desc),
			)
		case "fvobs":
			newFloat64ValueObserverRecord = true
			_ = metric.Must(meter).NewFloat64ValueObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					if newFloat64ValueObserverRecord {
						result.Observe(float64(val), keyValuePairs...)
						newFloat64ValueObserverRecord = false
					}
				},
				metric.WithDescription(desc),
			)
		default:
			invalidRecord = true
		}

		// Print a message based on whether a record was skipped or not.
		if invalidRecord {
			fmt.Printf("%v. [Skipped] Unsupported Record %v \n", i, record)
			invalidRecord = false
		} else {
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		}

		// Sleep for a while so the push controller won't push too much data at once.
		time.Sleep(pipelineOneSleepPeriod)
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
	fmt.Println("[Success] Created Config struct")

	// Run exporter setup pipeline.
	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(pushInterval))
	if err != nil {
		return nil, err
	}
	fmt.Println("[Success] Installed Exporter Pipeline")

	return pusher, nil
}

// initCSVReader creates a new instance of csv.Reader that enforces 3 fields per line.
func initCSVReader(filepath string) (*csv.Reader, error) {
	// Open the csv file to read from.
	data, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	// Create new reader that enforces 3 fields per line.
	reader := csv.NewReader(data)
	reader.FieldsPerRecord = 3

	return reader, err
}

// parsePipelineOneRecord parses a line from a csv file and extracts the instrument type,
// the value, and the key value pairs.
func parsePipelineOneRecord(record []string) (string, string, string, string, []kv.KeyValue, error) {
	// Parse the third field in the record for the key value pairs. The name and
	// description are ignored.
	stringFields := strings.Split(record[2], ",")
	numStringFields := len(stringFields)
	if numStringFields < 2 {
		return "", "", "", "", nil, fmt.Errorf("Missing name /description")
	}
	if numStringFields%2 != 0 {
		return "", "", "", "", nil, fmt.Errorf("Invalid key value pair")
	}

	name := stringFields[0]
	desc := stringFields[1]

	var keyValuePairs []kv.KeyValue
	for i := 2; i < numStringFields; i += 2 {
		keyValue := kv.String(stringFields[i], stringFields[i+1])
		keyValuePairs = append(keyValuePairs, keyValue)
	}

	return record[0], record[1], name, desc, keyValuePairs, nil
}

var newInt64SumObserverRecord bool = false
var newFloat64SumObserverRecord bool = false
var newInt64ValueObserverRecord bool = false
var newFloat64ValueObserverRecord bool = false
var newInt64UpDownSumObserverRecord bool = false
var newFloat64UpDownSumObserverRecord bool = false
