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
	"go.opentelemetry.io/contrib/exporters/metric/cortex/utils"
	"go.opentelemetry.io/otel/api/metric"
	"go.opentelemetry.io/otel/label"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/array"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/sum"
)

// runPipelineTwo runs a pipeline that creates checkpoint sets and exports
// them to Cortex by calling the exporter's Export() method.
func runPipelineTwo() {
	// Create exporter.
	exporter := initPipelineTwo()

	// Get context.
	ctx := context.Background()

	// Create CSV reader.
	reader := initPipelineTwoCSVReader()

	// Iterate through each line of data file.
	for i := 1; i > 0; i++ {
		// Retrieve the next line from the CSV file.
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// Retrieve necessary data for making a checkpoint set.
		aggType, values, name, labels, err := parsePipelineTwoRecord(record)
		if err != nil {
			log.Fatal(err)
		}

		// Create a checkpoint set based on the data.
		var checkpointSet *CheckpointSet
		switch aggType {
		case "sum":
			checkpointSet = buildCheckpointSet("sum", name, labels, values, metric.UpDownCounterKind)
		case "lval":
			checkpointSet = buildCheckpointSet("lval", name, labels, values, metric.ValueObserverKind)
		case "mmsc":
			checkpointSet = buildCheckpointSet("mmsc", name, labels, values, metric.ValueRecorderKind)
		case "dist":
			checkpointSet = buildCheckpointSet("dist", name, labels, values, metric.ValueRecorderKind)
		case "hist":
			checkpointSet = buildCheckpointSet("hist", name, labels, values, metric.ValueRecorderKind)
		}

		// Export to Cortex.
		err = exporter.Export(ctx, checkpointSet)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("[P2 Success] Parsed %v\n", record)
	}
}

// parsePipelineTwoRecord parses a line from a csv file and extracts the aggregation type,
// the values to update an instrument with, and the mock instrument's name / labels.
func parsePipelineTwoRecord(record []string) (string, []int64, string, []label.KeyValue, error) {
	// Aggregation type is the first field.
	aggType := record[0]

	// Retrieve the list of update values.
	var values []int64
	valuesStr := record[1]
	valuesStr = valuesStr[1 : len(valuesStr)-1] // Remove brackets
	valueFields := strings.Split(valuesStr, ",")
	for _, valueStr := range valueFields {
		value, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return "", nil, "", nil, err
		}
		values = append(values, value)
	}

	// Retrieve "instrument" name. Instrument doesn't actually exist since the pipelien
	// works with checkpoint sets directly.
	propertyFields := strings.Split(record[2], ",")
	name := propertyFields[0]

	// Retrieve the labels.
	var labels []label.KeyValue
	labelsStr := propertyFields[1]
	labelsStr = labelsStr[1 : len(labelsStr)-1] // Remove braces
	labelFields := strings.Split(labelsStr, ",")
	for _, pair := range labelFields {
		i := strings.Index(pair, ":")
		keyValue := label.String(pair[:i], pair[i+1:])
		labels = append(labels, keyValue)
	}

	return aggType, values, name, labels, nil
}

// buildCheckpointSet creates a new checkpoint set based on the aggregation type and adds
// different values to it.
func buildCheckpointSet(aggType string, name string, labels []label.KeyValue, values []int64, kind metric.Kind) *CheckpointSet {
	// Create sum checkpoint set with resource and descriptor.
	checkpointSet := NewCheckpointSet(nil)
	descriptor := metric.NewDescriptor(name, kind, metric.Int64NumberKind)

	// Create aggregation, add value, and update checkpoint set.
	var aggregation export.Aggregator
	var checkpoint export.Aggregator
	switch aggType {
	case "sum":
		aggregation, checkpoint = Unslice2(sum.New(2))
	case "lval":
		aggregation, checkpoint = Unslice2(lastvalue.New(2))
	case "mmsc":
		aggregation, checkpoint = Unslice2(minmaxsumcount.New(2, &descriptor))
	case "dist":
		aggregation, checkpoint = Unslice2(array.New(2))
	case "hist":
		boundaries := []float64{-25, 0, 25}
		aggregation, checkpoint = Unslice2(histogram.New(2, &descriptor, boundaries))
	}

	// Add different values to the checkpoint set.
	for _, value := range values {
		checkedUpdate(aggregation, metric.NewInt64Number(value), &descriptor)
	}
	err := aggregation.SynchronizedMove(checkpoint, &descriptor)
	if err != nil {
		log.Fatal(err)
	}
	checkpointSet.Add(time.Now(), &descriptor, checkpoint, labels...)

	return checkpointSet
}

// Performs the same range test the SDK does on behalf of the aggregator. This function is
// the same as CheckedUpdate() from the SDK, but it removes the testing.T pointer.
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

// initPipelineTwoCSVReader creates a new reader for reading and parsing csv files. It
// makes sure that each line has 3 records.
func initPipelineTwoCSVReader() *csv.Reader {
	// Open the csv file to read from.
	data, err := os.Open(pipelineTwoFilename)
	if err != nil {
		log.Fatal(err)
	}

	// Create new reader that enforces 3 fields per line.
	reader := csv.NewReader(data)
	reader.Comma = '|'
	reader.FieldsPerRecord = 3

	return reader
}

// initPipelineTwo creates a new Exporter for exporting metrics data to Cortex.
func initPipelineTwo() *cortex.Exporter {
	// Read config YAML file to generate a Config struct.
	config, err := utils.NewConfig("config.yml")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[P2 Success] Created Config struct")

	// Create an exporter.
	exporter, err := cortex.NewRawExporter(*config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("[P2 Success] Created New Cortex Exporter!")

	return exporter
}
