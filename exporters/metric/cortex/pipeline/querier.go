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

func storePipelineOneResults() error {
	file, err := os.Create(pipelineOneOutputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Open the csv file to read from.
	data, err := os.Open(pipelineOneFilename)
	if err != nil {
		return err
	}

	// Create new reader that enforces 3 fields per line.
	reader := csv.NewReader(data)
	reader.FieldsPerRecord = 3

	for i := 1; i > 0; i++ {
		var instrumentData *InstrumentData

		// Retrieve the next line from the CSV file.
		inputRecord, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		name := strings.Split(inputRecord[2], ",")[0]
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
		outputRecord := convertToRecord(instrumentData)
		file.WriteString(outputRecord + "\n")
	}
	return nil
}

func querySumInstrument(url string) (*InstrumentData, error) {
	// json := `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"name1956_sum","key1956":"value1956"},"value":[1598428477.909,"36"]}]}}`

	json, err := getJSON(url)
	if err != nil {
		return nil, err
	}

	instrumentData := InstrumentData{
		aggregation: "sum",
	}

	sum := gjson.Get(json, "data.result.0.value.1")
	metric := gjson.Get(json, "data.result.0.metric")
	name, labels := parseMetric(metric)
	instrumentData.name = name
	instrumentData.labels = labels
	instrumentData.sum = sum.Float()
	return &instrumentData, nil
}

func queryHistogramInstrument(url string) (*InstrumentData, error) {
	jsonSum, err := getJSON(url + "_sum")
	if err != nil {
		return nil, err
	}
	// jsonSum := `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"name1999_hist_sum","key1999":"value1999"},"value":[1598432608.412,"40"]}]}}`
	instrumentData := InstrumentData{
		aggregation: "histogram",
	}

	sum := gjson.Get(jsonSum, "data.result.0.value.1")
	metric := gjson.Get(jsonSum, "data.result.0.metric")
	name, labels := parseMetric(metric)
	instrumentData.name = name[:len(name)-4] // Name has an extra "_sum" at the end
	instrumentData.labels = labels
	instrumentData.sum = sum.Float()

	jsonCount, err := getJSON(url + "_count")
	if err != nil {
		return nil, err
	}
	count := gjson.Get(jsonCount, "data.result.0.value.1")
	instrumentData.count = count.Int()

	var buckets map[string]int64 = make(map[string]int64)
	jsonBuckets, err := getJSON(url)
	if err != nil {
		return nil, err
	}

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

func parseMetric(metric gjson.Result) (string, map[string]string) {
	var name string
	labels := make(map[string]string)

	metric.ForEach(func(key, value gjson.Result) bool {
		if key.Str == "__name__" {
			name = value.Str
			return true
		}
		labels[key.Str] = value.Str
		return true
	})
	return name, labels
}

type InstrumentData struct {
	name        string
	aggregation string
	sum         float64
	count       int64
	buckets     map[string]int64
	labels      map[string]string
}

func convertToRecord(data *InstrumentData) string {
	var record string
	var recordFields []string

	var labelFields []string
	for key, value := range data.labels {
		formatted := key + ":" + value
		labelFields = append(labelFields, formatted)
	}
	labels := "{" + strings.Join(labelFields, ",") + "}"
	properties := data.name + "," + labels

	if data.aggregation == "sum" {
		recordFields = []string{
			properties,
			"sum",
			strconv.FormatFloat(data.sum, 'f', -1, 64),
		}
		record = strings.Join(recordFields, "|")
	} else if data.aggregation == "histogram" {
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
