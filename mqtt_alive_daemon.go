package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"
)

type Config struct {
	MQTTBroker      string `yaml:"mqtt_broker"`
	MQTTTopic       string `yaml:"mqtt_topic"`
	MQTTUsername    string `yaml:"mqtt_username"`
	Command         string `yaml:"command"`
	Interval        int    `yaml:"interval"`
	ClientID        string `yaml:"client_id"`
	DiscoveryPrefix string `yaml:"discovery_prefix"`
	DeviceName      string `yaml:"device_name"`
}

type DiscoveryPayload struct {
	Name              string `json:"name"`
	UniqueID          string `json:"unique_id"`
	StateTopic        string `json:"state_topic"`
	PayloadOn         string `json:"payload_on"`
	PayloadOff        string `json:"payload_off"`
	DeviceClass       string `json:"device_class"`
	AvailabilityTopic string `json:"availability_topic"`
	Device            Device `json:"device"`
}

type Device struct {
	Identifiers  []string `json:"identifiers"`
	Name         string   `json:"name"`
	Manufacturer string   `json:"manufacturer"`
	Model        string   `json:"model"`
}

var client mqtt.Client
var config Config

func main() {
	log.Println("Starting MQTT Alive Daemon")

	// Read configuration
	config = readConfig()

	// Get MQTT password from environment variable
	mqttPassword := os.Getenv("MQTT_PASSWORD")
	if mqttPassword == "" {
		log.Fatal("MQTT_PASSWORD environment variable is not set")
	}

	// Create MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(config.MQTTBroker).
		SetClientID(config.ClientID).
		SetUsername(config.MQTTUsername).
		SetPassword(mqttPassword).
		SetWill(config.MQTTTopic+"/availability", "offline", 1, true)

	// Create MQTT client
	client = mqtt.NewClient(opts)

	// Connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	log.Println("Connected to MQTT broker:", config.MQTTBroker)

	// Publish discovery message
	publishDiscovery(client, config)

	// Publish initial availability
	client.Publish(config.MQTTTopic+"/availability", 1, true, "online")

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the power management listener
	powerChan := make(chan string)
	go listenForPowerEvents(powerChan)

	// Start the main loop
	go runMainLoop()

	// Wait for signals
	for {
		select {
		case sig := <-signalChan:
			log.Printf("Received signal: %v\n", sig)
			publishState("OFF")
			client.Disconnect(250)
			os.Exit(0)
		case event := <-powerChan:
			handlePowerEvent(event)
		}
	}
}

func runMainLoop() {
	for {
		state := "ON"
		if config.Command != "" {
			if err := runCommand(config.Command); err == nil {
				state = "ON"
			} else {
				state = "OFF"
			}
		}

		publishState(state)
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func publishState(state string) {
	token := client.Publish(config.MQTTTopic+"/state", 0, false, state)
	token.Wait()
	log.Printf("Published state: %s to topic: %s/state\n", state, config.MQTTTopic)
}

func handlePowerEvent(event string) {
	switch event {
	case "sleep":
		log.Println("System is going to sleep")
		publishState("OFF")
	case "wake":
		log.Println("System is waking up")
		publishState("ON")
	}
}

func listenForPowerEvents(powerChan chan<- string) {
	script := `
		#!/bin/bash
		on_sleep() {
			echo "sleep"
			exit 0
		}
		on_wake() {
			echo "wake"
			exit 0
		}
		trap on_sleep SIGTERM
		trap on_wake SIGCONT
		while true; do
			sleep 2 &
			wait $!
		done
	`

	cmd := exec.Command("bash", "-c", script)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start sleep/wake script: %v", err)
	}

	go func() {
		for {
			buf := make([]byte, 1024)
			n, err := stdout.Read(buf)
			if err != nil {
				log.Printf("Error reading script output: %v", err)
				return
			}
			output := strings.TrimSpace(string(buf[:n]))
			if output != "" {
				powerChan <- output
			}
		}
	}()

	// Start caffeinate to prevent sleep
	caffeinateCmd := exec.Command("caffeinate", "-d")
	if err := caffeinateCmd.Start(); err != nil {
		log.Fatalf("Failed to start caffeinate: %v", err)
	}

	go func() {
		err := caffeinateCmd.Wait()
		if err != nil {
			log.Printf("caffeinate exited: %v", err)
		}
	}()
}

func readConfig() Config {
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func runCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	return cmd.Run()
}

func publishDiscovery(client mqtt.Client, config Config) {
	payload := DiscoveryPayload{
		Name:              config.DeviceName,
		UniqueID:          config.ClientID,
		StateTopic:        config.MQTTTopic + "/state",
		PayloadOn:         "ON",
		PayloadOff:        "OFF",
		DeviceClass:       "connectivity",
		AvailabilityTopic: config.MQTTTopic + "/availability",
		Device: Device{
			Identifiers:  []string{config.ClientID},
			Name:         config.DeviceName,
			Manufacturer: "MQTT Alive Daemon",
			Model:        "v1.0",
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling discovery payload: %v", err)
		return
	}

	discoveryTopic := fmt.Sprintf("%s/binary_sensor/%s/config", config.DiscoveryPrefix, config.ClientID)
	token := client.Publish(discoveryTopic, 0, true, payloadJSON)
	token.Wait()

	log.Printf("Published discovery message to topic: %s\n", discoveryTopic)
	log.Printf("Discovery payload: %s\n", string(payloadJSON))
}
