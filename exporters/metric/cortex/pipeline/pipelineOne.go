package main

import (
	"bufio"
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
	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/sdk/metric/controller/push"
)

var creationIndices map[string]int = make(map[string]int)

// runPipelineOneInMemory runs a pipeline that records values to various instruments and
// exports metrics data to Cortex. The data is queried in batches and checked without
// writing anything to a file.
func runPipelineOneInMemory(
	inputFile string, answerFile string, batchSize int, numRecords int,
) {
	// Start a timer to measure how long the test takes.
	start := time.Now()
	fmt.Printf("[P1] Starting pipeline one test!\n\n")

	// Creates a push controller that calls Export() every 2 seconds.
	pusher, err := initPipeline(100 * time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}
	defer pusher.Stop()

	// Create a csv reader for reading the input data.
	reader, err := initCSVReader(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	// Open answers file to check expected results.
	file, err := os.Open(answerFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)

	// Create the meter for creating synchronous instruments.
	meter := pusher.Provider().Meter("example")
	ctx := context.Background()

	// Before starting the pipeline, wait a second to let everything start up.
	time.Sleep(1 * time.Second)

	// Create a new map for storing expected results and print a new line for formatting.
	resultMap := make(map[string]string, batchSize)
	fmt.Println()

	// Iterate through the CSV file line by line and record data to the instruments.
	for i := 1; i > 0; i++ {
		// Query each instrument in a batch.
		if len(resultMap)%batchSize == 0 && i != 1 {
			start := i - (batchSize + 1)
			end := i - 2
			fmt.Printf("\n[P1] Validating batch for lines %v to %v\n", start, end)

			// Check if there are any records were incorrect. Clear the map afterwards.
			if mismatches, valid := queryBatch(resultMap); !valid {
				fmt.Printf("[P1 Error] Found %v Mismatches in lines %v to %v\n", len(mismatches), start, end)
				for _, mismatch := range mismatches {
					fmt.Print(mismatch)
				}
				resultMap = make(map[string]string, batchSize)
				break
			}
			resultMap = make(map[string]string, batchSize)
			fmt.Printf("[P1] Validated batch for lines %v to %v\n\n", start, end)
		}

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
			_ = metric.Must(meter).NewInt64SumObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(int64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "fsobs":
			_ = metric.Must(meter).NewFloat64SumObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(float64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "iudobs":
			_ = metric.Must(meter).NewInt64UpDownSumObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(int64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "fudobs":
			_ = metric.Must(meter).NewFloat64UpDownSumObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(float64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "ivobs":
			_ = metric.Must(meter).NewInt64ValueObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(int64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "fvobs":
			_ = metric.Must(meter).NewFloat64ValueObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(float64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		}

		// Get the next line in the answers file.
		scanner.Scan()

		// Map the answer record to the instrument name.
		resultMap[name] = scanner.Text()
		elapsed := time.Since(start)
		fmt.Printf("[P1] Parsed %v/%v records. Elapsed time: %v\r", i, numRecords, elapsed)
	}

	// Print out elapsed time.
	elapsed := time.Since(start)
	fmt.Printf("[P1] Completed pipeline one test in %v!\n", elapsed)
}

// runPipelineOne runs a pipeline that records values to various instruments and exports
// metrics data to Cortex.
func runPipelineOne(filename string, delay time.Duration, numRecords int) {
	// Creates a push controller that calls Export() every 2 seconds.
	pusher, err := initPipeline(100 * time.Millisecond)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	defer pusher.Stop()

	// Create a csv reader for reading the input data.
	reader, err := initCSVReader(filename)
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
			_ = metric.Must(meter).NewInt64SumObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(int64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "fsobs":
			_ = metric.Must(meter).NewFloat64SumObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(float64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "iudobs":
			_ = metric.Must(meter).NewInt64UpDownSumObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(int64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "fudobs":
			_ = metric.Must(meter).NewFloat64UpDownSumObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(float64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "ivobs":
			_ = metric.Must(meter).NewInt64ValueObserver(
				name,
				func(_ context.Context, result metric.Int64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(int64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		case "fvobs":
			_ = metric.Must(meter).NewFloat64ValueObserver(
				name,
				func(_ context.Context, result metric.Float64ObserverResult) {
					creationIndices[name]++
					if creationIndices[name] <= 1 {
						result.Observe(float64(val), keyValuePairs...)
					}
				},
				metric.WithDescription(desc),
			)
		}

		fmt.Printf("%v. [P1] Parsed %v\n", i, record)

		// Sleep for a while so the push controller won't push too much data at once.
		time.Sleep(delay)
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
	fmt.Println("[P1] Created Config struct")

	// Run exporter setup pipeline.
	pusher, err := cortex.InstallNewPipeline(*config, push.WithPeriod(pushInterval))
	if err != nil {
		return nil, err
	}
	fmt.Println("[P1] Installed Exporter Pipeline")

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
func parsePipelineOneRecord(record []string) (string, string, string, string, []label.KeyValue, error) {
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

	var keyValuePairs []label.KeyValue
	for i := 2; i < numStringFields; i += 2 {
		keyValue := label.String(stringFields[i], stringFields[i+1])
		keyValuePairs = append(keyValuePairs, keyValue)
	}

	return record[0], record[1], name, desc, keyValuePairs, nil
}
