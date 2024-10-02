# Makefile for AgentFlow with Static Plugins in Separate Files

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=agentflow

MAIN_PACKAGE=.
LDFLAGS=-ldflags "-s -w"

all: deps build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PACKAGE)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME)

deps:
	$(GOMOD) download

fmt:
	gofmt -s -w .

run-flow: build
	./$(BINARY_NAME) start example-flow "User input here"

lint:
	golangci-lint run

.PHONY: all build test clean run deps fmt run-flow lint