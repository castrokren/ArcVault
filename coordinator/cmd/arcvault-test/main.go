package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	// Define subcommands
	loadCmd := flag.NewFlagSet("load", flag.ExitOnError)
	agents := loadCmd.Int("agents", 10, "Number of mock agents to spawn")
	jobsPerAgent := loadCmd.Int("jobs-per-agent", 20, "Number of jobs each agent executes")
	jobDuration := loadCmd.Int("job-duration", 5000, "Duration of each job in milliseconds")
	failureRate := loadCmd.Float64("failure-rate", 0, "Probability of job failure (0.0-1.0)")
	failureType := loadCmd.String("failure-type", "disconnect", "Type of failure to inject")
	output := loadCmd.String("output", "", "Output file for JSON report (empty = stdout)")

	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: arcvault-test <command> [options]\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  load   - Run load test with mock agents\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "load":
		loadCmd.Parse(os.Args[2:])
		runLoadTest(*agents, *jobsPerAgent, *jobDuration, *failureRate, *failureType, *output)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// runLoadTest executes a load test with the given parameters
func runLoadTest(agents, jobsPerAgent, jobDuration int, failureRate float64, failureType, outputFile string) {
	scenario := &LoadScenario{
		Agents:       agents,
		JobsPerAgent: jobsPerAgent,
		JobDurationMs: jobDuration,
		FailureRate:  failureRate,
		FailureType:  failureType,
	}

	harness := NewHarness(scenario)

	fmt.Printf("Starting load test: %d agents, %d jobs/agent, %dms per job\n",
		agents, jobsPerAgent, jobDuration)
	fmt.Println()

	report, err := harness.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running harness: %v\n", err)
		os.Exit(1)
	}

	// Output report
	if outputFile != "" {
		// Write JSON to file
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling report: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Report written to: %s\n", outputFile)
	}

	// Always print summary to console
	fmt.Println(report)
}
