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

func main() {
	// Start a timer to measure how long pipeline test takes.
	start := time.Now()
	fmt.Printf("Starting pipeline test!\n\n")

	// Creates a push controller that calls Export() every 2 seconds.
	pusher, err := initPipeline(2 * time.Second)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer pusher.Stop()

	// Create a csv reader for reading the input data.
	reader, err := initCSVReader("data.csv")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	// Create the meter for creating instruments.
	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	// Create synchronous instruments. Async instruments need to be created each time
	// because the Observe() method can only be called in the callback function.
	int64Counter, float64Counter, int64UpDownCounter, float64UpDownCounter, int64ValueRecorder, float64ValueRecorder := initSyncInstruments(meter)
	fmt.Printf("[Success] Created instruments!\n\n")

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

		// Parse the line into a Data struct.
		data, err := parseRecord(record)
		if err != nil {
			log.Fatal(err)
		}

		// All values in the generated data are integers, but are read as strings.
		val, err := strconv.Atoi(data.value)
		if err != nil {
			log.Fatalf("Failed to parse %v as an integer", data.value)
		}

		// Record the data in the correct instrument.
		switch data.instrument {
		case "ictr", "sctr":
			int64Counter.Add(ctx, int64(val), data.keyValuePairs...)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "fctr", "dctr":
			float64Counter.Add(ctx, float64(val), data.keyValuePairs...)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "iudctr", "sudctr":
			int64UpDownCounter.Add(ctx, int64(val), data.keyValuePairs...)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "fudctr", "dudctr":
			float64UpDownCounter.Add(ctx, float64(val), data.keyValuePairs...)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "ivrec", "svrec":
			int64ValueRecorder.Record(ctx, int64(val), data.keyValuePairs...)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "fvrec", "dvrec":
			float64ValueRecorder.Record(ctx, float64(val), data.keyValuePairs...)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "isobs", "ssobs":
			_ = metric.Must(meter).NewInt64SumObserver(
				"pipeline.int64SumObserver",
				func(_ context.Context, result metric.Int64ObserverResult) {
					result.Observe(int64(val), data.keyValuePairs...)
				},
				metric.WithDescription("Asynchronous additive monotonic instrument for 64-bit integers"),
			)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "fsobs", "dsobs":
			_ = metric.Must(meter).NewFloat64SumObserver(
				"pipeline.float64SumObserver",
				func(_ context.Context, result metric.Float64ObserverResult) {
					result.Observe(float64(val), data.keyValuePairs...)
				},
				metric.WithDescription("Asynchronous additive monotonic instrument for 64-bit integers"),
			)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "iudobs", "sudobs":
			_ = metric.Must(meter).NewInt64UpDownSumObserver(
				"pipeline.int64UpDownSumObserver",
				func(_ context.Context, result metric.Int64ObserverResult) {
					result.Observe(int64(val), data.keyValuePairs...)
				},
				metric.WithDescription("Asynchronous additive instrument for 64-bit integers"),
			)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "fudobs", "dudobs":
			_ = metric.Must(meter).NewFloat64UpDownSumObserver(
				"pipeline.float64UpDownSumObserver",
				func(_ context.Context, result metric.Float64ObserverResult) {
					result.Observe(float64(val), data.keyValuePairs...)
				},
				metric.WithDescription("Asynchronous additive monotonic instrument for 64-bit integers"),
			)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "ivobs", "svobs":
			_ = metric.Must(meter).NewInt64ValueObserver(
				"pipeline.int64ValueObserver",
				func(_ context.Context, result metric.Int64ObserverResult) {
					result.Observe(int64(val), data.keyValuePairs...)
				},
				metric.WithDescription("Non-additive asynchronous instrument for 64-bit integers"),
			)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		case "fvobs", "dvobs":
			_ = metric.Must(meter).NewFloat64ValueObserver(
				"pipeline.float64ValueObserver",
				func(_ context.Context, result metric.Float64ObserverResult) {
					result.Observe(float64(val), data.keyValuePairs...)
				},
				metric.WithDescription("Non-additive asynchronous instrument for 64-bit floats"),
			)
			fmt.Printf("%v. [Success] Parsed %v\n", i, record)
		default:
			fmt.Printf("%v. [Skipped] Unsupported Record %v \n", i, record)
		}

		// Sleep for a while so the push controller won't push too much data at once.
		time.Sleep(2 * time.Second)
	}

	// Print out elapsed time.
	elapsed := time.Since(start)
	fmt.Printf("\n[Success] Completed pipeline test!\n")
	fmt.Printf("Elapsed Time: %v\n", elapsed)
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

// Data is the parsed data from a single line in the CSV file.
type Data struct {
	instrument    string
	value         string
	name          string
	description   string
	keyValuePairs []kv.KeyValue
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

// parseRecord reads a slice of fields and extracts information about a instrument from
// it.
func parseRecord(record []string) (*Data, error) {
	// Parse the third field in the record for the name, description, and key value pairs.
	stringFields := strings.Split(record[2], ",")
	numStringFields := len(stringFields)
	if numStringFields < 2 {
		return nil, fmt.Errorf("Missing name /description")
	}
	if numStringFields%2 != 0 {
		return nil, fmt.Errorf("Invalid key value pair")
	}

	name := stringFields[0]
	description := stringFields[1]
	var keyValuePairs []kv.KeyValue

	for i := 2; i < numStringFields; i += 2 {
		keyValue := kv.String(stringFields[i], stringFields[i+1])
		keyValuePairs = append(keyValuePairs, keyValue)
	}

	// Create and return a struct with info on a single csv line.
	line := &Data{
		instrument:    record[0],
		value:         record[1],
		name:          name,
		description:   description,
		keyValuePairs: keyValuePairs,
	}

	return line, nil
}

// initSyncInstruments creates and returns int64 and float64 instances of all 3
// synchronous instruments.
func initSyncInstruments(meter metric.Meter) (
	metric.Int64Counter, metric.Float64Counter, metric.Int64UpDownCounter, metric.Float64UpDownCounter, metric.Int64ValueRecorder, metric.Float64ValueRecorder,
) {
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

	return int64Counter, float64Counter, int64UpDownCounter,
		float64UpDownCounter, int64ValueRecorder, float64ValueRecorder
}
