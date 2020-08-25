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
var pipelineOneFilename string = "test.csv"
var pipelineOneSleepPeriod time.Duration = 1 * time.Second

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

	// Create synchronous instruments. Async instruments need to be created each time
	// because the Observe() method can only be called in the callback function.
	int64Counter, float64Counter, int64UpDownCounter, float64UpDownCounter, int64ValueRecorder, float64ValueRecorder := initSyncInstruments(meter)
	fmt.Printf("[Success] Created synchronous instruments!\n\n")

	initAsyncInstruments(meter)
	fmt.Printf("[Success] Created asynchronous instruments!\n\n")

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
		instrument, valueStr, keyValuePairs, err := parsePipelineOneRecord(record)
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
			int64Counter.Add(ctx, int64(val), keyValuePairs...)
		case "fctr":
			float64Counter.Add(ctx, float64(val), keyValuePairs...)
		case "iudctr":
			int64UpDownCounter.Add(ctx, int64(val), keyValuePairs...)
		case "fudctr":
			float64UpDownCounter.Add(ctx, float64(val), keyValuePairs...)
		case "ivrec":
			int64ValueRecorder.Record(ctx, int64(val), keyValuePairs...)
		case "fvrec":
			float64ValueRecorder.Record(ctx, float64(val), keyValuePairs...)
		case "isobs":
			int64SumObserverData.value = int64(val)
			int64SumObserverData.kvPairs = keyValuePairs
		case "fsobs":
			float64SumObserverData.value = float64(val)
			float64SumObserverData.kvPairs = keyValuePairs
		case "iudobs":
			int64UpDownSumObserverData.value = int64(val)
			int64UpDownSumObserverData.kvPairs = keyValuePairs
		case "fudobs":
			float64UpDownSumObserverData.value = float64(val)
			float64UpDownSumObserverData.kvPairs = keyValuePairs
		case "ivobs":
			int64ValueObserverData.value = int64(val)
			int64ValueObserverData.kvPairs = keyValuePairs
		case "fvobs":
			float64ValueObserverData.value = float64(val)
			float64ValueObserverData.kvPairs = keyValuePairs
		default:
			invalidRecord = true
		}

		// Print a message based on whether a record was skipped or not.
		if invalidRecord {
			fmt.Printf("%v. [Skipped] Unsupported Record %v \n", i, record)
			invalidRecord = false
		} else {
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
			parsedNewRecord = true
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
func parsePipelineOneRecord(record []string) (string, string, []kv.KeyValue, error) {
	// Parse the third field in the record for the key value pairs. The name and
	// description are ignored.
	stringFields := strings.Split(record[2], ",")
	numStringFields := len(stringFields)
	if numStringFields < 2 {
		return "", "", nil, fmt.Errorf("Missing name /description")
	}
	if numStringFields%2 != 0 {
		return "", "", nil, fmt.Errorf("Invalid key value pair")
	}

	var keyValuePairs []kv.KeyValue
	for i := 2; i < numStringFields; i += 2 {
		keyValue := kv.String(stringFields[i], stringFields[i+1])
		keyValuePairs = append(keyValuePairs, keyValue)
	}

	return record[0], record[1], keyValuePairs, nil
}

// initSyncInstruments creates and returns int64 and float64 instances of all 3
// synchronous instruments.
func initSyncInstruments(meter metric.Meter) (
	metric.Int64Counter, metric.Float64Counter,
	metric.Int64UpDownCounter, metric.Float64UpDownCounter,
	metric.Int64ValueRecorder, metric.Float64ValueRecorder,
) {
	int64Counter := metric.Must(meter).NewInt64Counter("int64Counter")

	float64Counter := metric.Must(meter).NewFloat64Counter("float64Counter")

	int64UpDownCounter := metric.Must(meter).NewInt64UpDownCounter("int64UpDownCounter")

	float64UpDownCounter := metric.Must(meter).NewFloat64UpDownCounter("float64UpDownCounter")

	int64ValueRecorder := metric.Must(meter).NewInt64ValueRecorder("int64ValueRecorder")

	float64ValueRecorder := metric.Must(meter).NewFloat64ValueRecorder("float64ValueRecorder")

	return int64Counter, float64Counter, int64UpDownCounter,
		float64UpDownCounter, int64ValueRecorder, float64ValueRecorder
}

// initAsyncInstruments creates and returns int64 and float64 instances of all 3
// synchronous instruments.
func initAsyncInstruments(meter metric.Meter) {
	_ = metric.Must(meter).NewInt64SumObserver(
		"int64SumObserver",
		func(_ context.Context, result metric.Int64ObserverResult) {
			if parsedNewRecord {
				result.Observe(
					int64SumObserverData.value,
					int64SumObserverData.kvPairs...,
				)
				parsedNewRecord = false
			}
		},
	)

	_ = metric.Must(meter).NewFloat64SumObserver(
		"float64SumObserver",
		func(_ context.Context, result metric.Float64ObserverResult) {
			if parsedNewRecord {
				result.Observe(
					float64SumObserverData.value,
					float64SumObserverData.kvPairs...,
				)
				parsedNewRecord = false
			}
		},
	)

	_ = metric.Must(meter).NewInt64UpDownSumObserver(
		"int64UpDownSumObserver",
		func(_ context.Context, result metric.Int64ObserverResult) {
			if parsedNewRecord {
				result.Observe(
					int64UpDownSumObserverData.value,
					int64UpDownSumObserverData.kvPairs...,
				)
			}
		},
	)

	_ = metric.Must(meter).NewFloat64UpDownSumObserver(
		"float64UpDownSumObserver",
		func(_ context.Context, result metric.Float64ObserverResult) {
			if parsedNewRecord {
				result.Observe(
					float64UpDownSumObserverData.value,
					float64UpDownSumObserverData.kvPairs...,
				)
				parsedNewRecord = false
			}
		},
	)

	_ = metric.Must(meter).NewInt64ValueObserver(
		"int64ValueObserver",
		func(_ context.Context, result metric.Int64ObserverResult) {
			if parsedNewRecord {
				result.Observe(
					int64ValueObserverData.value,
					int64ValueObserverData.kvPairs...,
				)
				parsedNewRecord = false
			}
		},
	)

	_ = metric.Must(meter).NewFloat64ValueObserver(
		"float64ValueObserver",
		func(_ context.Context, result metric.Float64ObserverResult) {
			if parsedNewRecord {
				result.Observe(
					float64ValueObserverData.value,
					float64ValueObserverData.kvPairs...,
				)
				parsedNewRecord = false
			}
		},
	)
}

// Structs for instrument data. Async instruments can only record values inside the
// callback function, which cannot be accessed after creating the instrument. To get
// around this, the callback functions will record values from these structs. The structs'
// values will change as more csv records are read and parsed.
var int64ValueObserverData struct {
	value   int64
	kvPairs []kv.KeyValue
}

var float64ValueObserverData struct {
	value   float64
	kvPairs []kv.KeyValue
}

var int64SumObserverData struct {
	value   int64
	kvPairs []kv.KeyValue
}

var float64SumObserverData struct {
	value   float64
	kvPairs []kv.KeyValue
}

var int64UpDownSumObserverData struct {
	value   int64
	kvPairs []kv.KeyValue
}

var float64UpDownSumObserverData struct {
	value   float64
	kvPairs []kv.KeyValue
}

// This boolean indicates that a new record has been parsed and that an async instrument
// now has a new value to record.
var parsedNewRecord bool = false
