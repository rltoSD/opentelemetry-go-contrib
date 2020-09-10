package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"
)

// Settings.
var pipelineTwoFilename string = "data/PrometheusDataSecond.csv"
var pipelineTwoOutputFile string = "data/pipelineTwoResults.csv"

func main() {
	// // Run and validate pipeline one in-memory.
	// runPipelineOneInMemory(
	// 	"data/PrometheusDataFirst.csv",
	// 	"data/PrometheusAnswersFirst.csv",
	// 	1000,
	// 	5000,
	// )

	// Start a timer to measure how long pipeline test takes.
	start := time.Now()

	// Run PipelineOne test
	fmt.Printf("[P1] Starting pipeline one test!\n\n")
	fmt.Printf("[P1] Exporting data to Cortex!\n")
	runPipelineOne("data/PrometheusDataFirst.csv", 0*time.Millisecond, 5000)

	fmt.Printf("\n[P1] Querying data from Cortex and writing results to disk!\n")
	storePipelineOneResults("data/PrometheusDataFirst.csv", "data/pipelineOneResults.csv")

	fmt.Printf("\n[P1] Comparing the results and answers files!\n")
	validatePipelineOne("data/pipelineOneResults.csv")

	// // Export to Cortex.
	// fmt.Printf("[P2] Exporting data to Cortex!\n")
	// runPipelineTwo()

	// // Query Cortex and write results to `pipelineOneOutputFile`.
	// fmt.Printf("\n[P2] Querying data from Cortex and writing results to disk!\n")
	// storePipelineTwoResults()

	// // Validate that the results file and the answers file are the same.
	// fmt.Printf("\n[P2] Comparing the results and answers files!\n")
	// validatePipelineTwo()

	// Print out elapsed time.
	elapsed := time.Since(start)
	fmt.Printf("\n[Success] Completed pipeline tests!\n")
	fmt.Printf("Elapsed Time: %v\n", elapsed)
}

// validatePipelineOne opens and compares the results and answers file for pipeline one.
// It prints whether the files are the same and removes the results file if it is.
func validatePipelineOne(filename string) {
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
		os.Remove(filename)
	} else {
		fmt.Println("[Failure] Pipeline One Validation Failed. Check files.")
	}
}

// validatePipelineTwo opens and compares the results and answers file for pipeline one.
// It prints whether the files are the same and removes the results file if it is.
func validatePipelineTwo() {
	results, err := ioutil.ReadFile("data/pipelineTwoResults.csv")
	if err != nil {
		log.Fatal(err)
	}

	answers, err := ioutil.ReadFile("data/PrometheusAnswersSecond.csv")
	if err != nil {
		log.Fatal(err)
	}

	equal := bytes.Equal(results, answers)

	if equal {
		fmt.Println("[Success] Pipeline Two Validation Succeeded.")
		os.Remove(pipelineTwoOutputFile)
	} else {
		fmt.Println("[Failure] Pipeline Two Validation Failed. Check files.")
	}
}
