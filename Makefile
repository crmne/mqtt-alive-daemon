# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=mqtt-alive-daemon

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
else
    DETECTED_OS := $(shell uname -s)
endif

# Set OS-specific variables
ifeq ($(DETECTED_OS),Darwin)
    INSTALL_DIR=/usr/local/bin
    CONFIG_DIR=/usr/local/etc/mqtt-alive-daemon
    SERVICE_DIR=/Library/LaunchDaemons
    SERVICE_FILE=me.paolino.mqtt-alive-daemon.plist
else ifeq ($(DETECTED_OS),Linux)
    INSTALL_DIR=/usr/local/bin
    CONFIG_DIR=/etc/mqtt-alive-daemon
    SERVICE_DIR=/etc/systemd/system
    SERVICE_FILE=mqtt-alive-daemon.service
endif

all: build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

install: build
	@echo "This operation requires root privileges. Please enter your password if prompted."
	sudo mkdir -p $(INSTALL_DIR)
	sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	sudo mkdir -p $(CONFIG_DIR)
	sudo chmod 755 $(CONFIG_DIR)
	sudo cp -n config.yaml.example $(CONFIG_DIR)/config.yaml || true
	@echo "Example configuration file copied to $(CONFIG_DIR)/config.yaml"
	@echo "!!! Please edit $(CONFIG_DIR)/config.yaml with your MQTT broker details and desired settings."
	sudo mkdir -p $(SERVICE_DIR)
ifeq ($(DETECTED_OS),Darwin)
	sudo sed 's|/path/to/your/mqtt-alive-daemon|$(INSTALL_DIR)/$(BINARY_NAME)|g' $(SERVICE_FILE) | sudo tee $(SERVICE_DIR)/$(SERVICE_FILE) > /dev/null
	sudo launchctl unload $(SERVICE_DIR)/$(SERVICE_FILE)
	sudo launchctl load $(SERVICE_DIR)/$(SERVICE_FILE)
	@echo "LaunchDaemon installed and loaded."
else ifeq ($(DETECTED_OS),Linux)
	sudo sed 's|/path/to/your/mqtt-alive-daemon|$(INSTALL_DIR)/$(BINARY_NAME)|g' $(SERVICE_FILE) | sudo tee $(SERVICE_DIR)/$(SERVICE_FILE) > /dev/null
	sudo systemctl daemon-reload
	sudo systemctl enable mqtt-alive-daemon
	sudo systemctl restart mqtt-alive-daemon
	@echo "Systemd service installed and started."
endif
	@echo "Installation complete!"

uninstall:
	@echo "This operation requires root privileges. Please enter your password if prompted."
ifeq ($(DETECTED_OS),Darwin)
	sudo launchctl unload $(SERVICE_DIR)/$(SERVICE_FILE)
	sudo rm $(SERVICE_DIR)/$(SERVICE_FILE)
else ifeq ($(DETECTED_OS),Linux)
	sudo systemctl stop mqtt-alive-daemon
	sudo systemctl disable mqtt-alive-daemon
	sudo rm $(SERVICE_DIR)/$(SERVICE_FILE)
	sudo systemctl daemon-reload
endif
	sudo rm -rf $(CONFIG_DIR)
	sudo rm $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Uninstallation complete!"

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

run: build
	sudo ./$(BINARY_NAME)

deps:
	$(GOGET) -v -d ./...

.PHONY: all build install uninstall clean test run deps
