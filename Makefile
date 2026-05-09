SHELL := /bin/sh

.PHONY: build test fmt run-worker run-controller run-cli-s1 run-cli-s2 run-cli-s3 run-cli-status

build:
	go build ./...

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

run-worker:
	go run ./cmd/worker

run-controller:
	go run ./cmd/controller

run-cli-s1:
	go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/registration_storm.yaml

run-cli-s2:
	go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/call_setup_rate.yaml

run-cli-s3:
	go run ./cmd/cli run -controller http://localhost:8080 -scenario scenarios/media_stress.yaml

run-cli-status:
	go run ./cmd/cli status -controller http://localhost:8080
