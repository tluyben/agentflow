# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=agentflow
BINARY_UNIX=$(BINARY_NAME)_unix

# Main package path
MAIN_PACKAGE=.

# Linter
GOLINT=golangci-lint

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PACKAGE)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

run: build
	./$(BINARY_NAME)

deps:
	$(GOGET) -v -d ./...
	$(GOMOD) tidy

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PACKAGE)

lint:
	$(GOLINT) run

# Database operations
index:
	./$(BINARY_NAME) index

search:
	@read -p "Enter search query: " query; \
	./$(BINARY_NAME) search "$$query" start

# Docker
docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run -it --rm --name running-$(BINARY_NAME) $(BINARY_NAME)

.PHONY: all build test clean run deps build-linux lint index search docker-build docker-run