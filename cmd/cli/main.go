package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
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
	out, _ := io.ReadAll(resp.Body)
	fmt.Printf("status: %s\n%s\n", resp.Status, string(out))
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
	out, _ := io.ReadAll(resp.Body)
	fmt.Printf("status: %s\n%s\n", resp.Status, string(out))
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  cli run -scenario scenarios/registration_storm.yaml [-controller http://localhost:8080]")
	fmt.Println("  cli stop [-controller http://localhost:8080]")
}
