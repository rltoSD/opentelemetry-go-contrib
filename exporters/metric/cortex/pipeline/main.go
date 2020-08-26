package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var pipelineOneFilename string = "data/PrometheusDataFirst.csv"
var pipelineOneSleepPeriod time.Duration = 0 * time.Millisecond
var pipelineOneOutputFile string = "data/pipelineOneResults.csv"

func main() {
	// Start a timer to measure how long pipeline test takes.
	start := time.Now()
	fmt.Printf("Starting pipeline test!\n\n")

	// Export to Cortex.
	fmt.Printf("Exporting data to Cortex!\n")
	runPipelineOne()

	// Query Cortex and write results to `pipelineOneOutputFile`.
	fmt.Printf("\nQuerying data from Cortex and writing results to disk!\n")
	storePipelineOneResults()

	// Validate that the results file and the answers file are the same.
	fmt.Printf("\nComparing the results and answers files!\n")
	validatePipelineOne()

	// Print out elapsed time.
	elapsed := time.Since(start)
	fmt.Printf("\n[Success] Completed pipeline test!\n")
	fmt.Printf("Elapsed Time: %v\n", elapsed)
}

func validatePipelineOne() {
	results, err := ioutil.ReadFile("data/pipelineOneResults.csv")
	if err != nil {
		log.Fatal(err)
	}

	answers, err := ioutil.ReadFile("data/PrometheusAnswersFirst.csv")
	if err != nil {
		log.Fatal(err)
	}

	equal := bytes.Equal(results, answers)

	if equal {
		fmt.Println("[Success] Pipeline One Validation Succeeded.")
		os.Remove(pipelineOneOutputFile)
	} else {
		fmt.Println("[Failure] Pipeline One Validation Failed. Check files.")
	}
}
