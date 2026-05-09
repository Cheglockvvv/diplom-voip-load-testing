package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"diplom_code/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}
	switch os.Args[1] {
	case "run":
		runCmd(os.Args[2:])
	case "stop":
		stopCmd(os.Args[2:])
	case "status":
		statusCmd(os.Args[2:])
	default:
		usage()
	}
}

func runCmd(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	controller := fs.String("controller", "http://localhost:8080", "Controller URL")
	scenarioPath := fs.String("scenario", "", "Path to YAML scenario")
	fs.Parse(args)

	if *scenarioPath == "" {
		fmt.Println("scenario path is required")
		os.Exit(1)
	}

	sc, err := config.LoadScenario(*scenarioPath)
	if err != nil {
		fmt.Printf("failed to load scenario: %v\n", err)
		os.Exit(1)
	}
	if err := sc.Validate(); err != nil {
		fmt.Printf("scenario validation error: %v\n", err)
		os.Exit(1)
	}
	body, _ := json.Marshal(sc)
	resp, err := http.Post(*controller+"/run", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Printf("failed to call controller: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func stopCmd(args []string) {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	controller := fs.String("controller", "http://localhost:8080", "Controller URL")
	fs.Parse(args)

	resp, err := http.Post(*controller+"/stop", "text/plain", nil)
	if err != nil {
		fmt.Printf("failed to call controller: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func statusCmd(args []string) {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	controller := fs.String("controller", "http://localhost:8080", "Controller URL")
	fs.Parse(args)

	resp, err := http.Get(*controller + "/status")
	if err != nil {
		fmt.Printf("failed to call controller: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	printResponse(resp)
}

func printResponse(resp *http.Response) {
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		fmt.Printf("status: %s\n", resp.Status)
		return
	}
	fmt.Printf("status: %s\n", resp.Status)
	for _, k := range []string{"run_id", "state", "message"} {
		if v, ok := payload[k]; ok {
			fmt.Printf("%s: %v\n", k, v)
		}
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  cli run -scenario scenarios/registration_storm.yaml [-controller http://localhost:8080]")
	fmt.Println("  cli stop [-controller http://localhost:8080]")
	fmt.Println("  cli status [-controller http://localhost:8080]")
}
