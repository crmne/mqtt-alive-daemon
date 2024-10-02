# Version
COMMIT=$(shell git rev-parse HEAD)
DATE=$(shell date -u +%Y-%m-%d)
# Go parameters
GOCMD=go 
GOBUILD=$(GOCMD) build -buildvcs=false -ldflags "-X main.Commit=$(COMMIT) -X main.Date=$(DATE)"
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=mqtt-alive-daemon
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v
test:
	$(GOTEST) -v ./...
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)
deps:
	$(GOGET) github.com/eclipse/paho.mqtt.golang
	$(GOGET) github.com/denisbrodbeck/machineid
	$(GOGET) gopkg.in/yaml.v2

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v
