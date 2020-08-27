package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// InstrumentData holds relevant information about a single instrument.
type InstrumentData struct {
	name        string
	aggregation string
	sum         float64
	count       int64
	buckets     map[string]int64
	labels      map[string]string
}

// storePipelineOneResults iterates through a generated data file, queries Cortex for each
// line in the file, converts the response to a csv record, and then writes that record to
// a new file.
func storePipelineOneResults() error {
	// Open a file to write the results to.
	file, err := os.Create(pipelineOneOutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Open the generated data csv file to read from.
	data, err := os.Open(pipelineOneFilename)
	if err != nil {
		return err
	}

	// Create new reader that enforces 3 fields per line.
	reader := csv.NewReader(data)
	reader.FieldsPerRecord = 3

	// Iterate through each line of the data csv file.
	for {
		// Retrieve the next line from the CSV file and exit the loop when there are no
		// more lines or if there was an error.
		inputRecord, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		// Parse the data record and retrieve the name of the instrument.
		name := strings.Split(inputRecord[2], ",")[0]

		// Make a Cortex instant query for the instrument using the name and store the
		// response as a InstrumentData struct.
		var instrumentData *InstrumentData
		url := "http://0.0.0.0:9009/api/prom/api/v1/query?query=" + name
		if strings.Contains(name, "_sum") {
			instrumentData, err = querySumInstrument(url)
			if err != nil {
				log.Fatal(err)
			}
		} else if strings.Contains(name, "_hist") {
			instrumentData, err = queryHistogramInstrument(url)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Convert the InstrumentData struct into a csv record in the same format as the
		// generated answers file.
		outputRecord := convertToRecord(instrumentData)

		// Write the record to the file.
		file.WriteString(outputRecord + "\n")
	}
	return nil
}

// querySumInstrument queries Cortex for an instrument that uses the Sum aggregation.
// Only the name, labels, and sum properties will be filled.
func querySumInstrument(url string) (*InstrumentData, error) {
	// Create a sum aggregation InstrumentData struct.
	instrumentData := InstrumentData{
		aggregation: "sum",
	}

	// Retrieve the JSON response from Cortex.
	json, err := getJSON(url)
	if err != nil {
		return nil, err
	}

	// Retrieve sum from JSON.
	sum := gjson.Get(json, "data.result.0.value.1")

	// Retrieve the name and labels. They are stored in a `metric` JSON object.
	metric := gjson.Get(json, "data.result.0.metric")
	name, labels := parseMetric(metric)

	// Set the struct properties.
	instrumentData.name = name
	instrumentData.labels = labels
	instrumentData.sum = sum.Float()

	return &instrumentData, nil
}

// queryHistogramInstrument queries Cortex for an instrument that uses the Histogram
// aggregation. Histograms are exported as 3 different TimeSeries, so there will be 3
// different HTTP GET requests, one each for the sum, count, and buckets.
func queryHistogramInstrument(url string) (*InstrumentData, error) {
	// Create a histogram aggregation InstrumentData struct.
	instrumentData := InstrumentData{
		aggregation: "histogram",
	}

	// Retrieve sum JSON. The exporter exports Histogram sum data as a TimeSeries with the
	// name as <name>_sum.
	jsonSum, err := getJSON(url + "_sum")
	if err != nil {
		return nil, err
	}

	// Retrieve the sum from the JSON.
	sum := gjson.Get(jsonSum, "data.result.0.value.1")

	// Retrieve the names and labels. The name and labels are common to all three 3
	// requests, so it is done here. Note that the "le" label is ignored by the answers
	// file, which is why the labels can be gathered with the sum json.
	metric := gjson.Get(jsonSum, "data.result.0.metric")
	name, labels := parseMetric(metric)

	// Set the struct properties. Note that the instrument name from this JSON has an
	// additional "_sum", so it is removed using substrings.
	instrumentData.name = name[:len(name)-4]
	instrumentData.labels = labels
	instrumentData.sum = sum.Float()

	// Retrieve the count JSON.
	jsonCount, err := getJSON(url + "_count")
	if err != nil {
		return nil, err
	}

	// Retrieve and set the count.
	count := gjson.Get(jsonCount, "data.result.0.value.1")
	instrumentData.count = count.Int()

	// Retrieve the buckets JSON. There are
	var buckets map[string]int64 = make(map[string]int64)
	jsonBuckets, err := getJSON(url)
	if err != nil {
		return nil, err
	}

	// Iterate through the results object, which contains objects for each bucket, and
	// store the bucket count value in the `buckets` dictionary.
	results := gjson.Get(jsonBuckets, "data.result")
	results.ForEach(func(key, value gjson.Result) bool {
		metricValue := gjson.Parse(value.String()).Get("value.1").Int()
		metricBoundary := gjson.Parse(value.String()).Get("metric.le").String()
		buckets[metricBoundary] = metricValue
		return true
	})
	instrumentData.buckets = buckets

	return &instrumentData, nil
}

// getJSON makes a HTTP GET request to Cortex and returns a JSON as a string.
func getJSON(url string) (string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status code: %v", res.StatusCode)
	}

	// Convert the response body into a JSON string.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// parseMetric iterates through a JSON object representing a single metric and returns the
// name and the labels in it.
func parseMetric(metric gjson.Result) (string, map[string]string) {
	var name string
	labels := make(map[string]string)

	metric.ForEach(func(key, value gjson.Result) bool {
		// Everything other `__name__` is a label.
		if key.Str == "__name__" {
			name = value.Str
			return true
		}
		labels[key.Str] = value.Str
		return true
	})
	return name, labels
}

// convertToRecord converts a InstrumentData struct to a formatted csv record string that
// will be printed to the results file.
func convertToRecord(data *InstrumentData) string {
	var record string
	var recordFields []string

	// Parse the labels and store them in curly braces.
	var labelFields []string
	for key, value := range data.labels {
		formatted := key + ":" + value
		labelFields = append(labelFields, formatted)
	}
	labels := "{" + strings.Join(labelFields, ",") + "}"
	properties := data.name + "," + labels

	// Create the record string depending on the aggregation type.
	if data.aggregation == "sum" {
		recordFields = []string{
			properties,
			"sum",
			strconv.FormatFloat(data.sum, 'f', -1, 64),
		}
		record = strings.Join(recordFields, "|")
	} else if data.aggregation == "histogram" {
		// Values are hard-coded for now since order is not guaranteed in a map.
		bucketFields := []string{
			strconv.FormatInt(data.buckets["-25"], 10),
			strconv.FormatInt(data.buckets["0"], 10),
			strconv.FormatInt(data.buckets["25"], 10),
			strconv.FormatInt(data.buckets["+inf"], 10),
		}
		buckets := "{" + strings.Join(bucketFields, ",") + "}"
		recordFields = []string{
			properties,
			"histogram",
			strconv.FormatFloat(data.sum, 'f', -1, 64),
			strconv.FormatInt(data.count, 10),
			buckets,
		}
		record = strings.Join(recordFields, "|")
	}
	return record
}
