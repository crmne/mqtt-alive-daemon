package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
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
	PayloadOnline     string `json:"payload_online"`
	PayloadOffline    string `json:"payload_offline"`
	DeviceClass       string `json:"device_class"`
	AvailabilityTopic string `json:"availability_topic"`
}

func main() {
	log.Println("Starting MQTT Alive Daemon")

	// Read configuration
	config := readConfig()

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
	client := mqtt.NewClient(opts)

	// Connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	log.Println("Connected to MQTT broker:", config.MQTTBroker)

	// Publish discovery message
	publishDiscovery(client, config)

	// Publish initial availability
	client.Publish(config.MQTTTopic+"/availability", 1, true, "online")

	for {
		message := "alive"
		if config.Command != "" {
			if err := runCommand(config.Command); err == nil {
				message = "command_success"
			} else {
				message = "command_failure"
			}
		}

		token := client.Publish(config.MQTTTopic+"/state", 0, false, message)
		token.Wait()

		log.Printf("Published message: %s to topic: %s/state\n", message, config.MQTTTopic)

		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
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
		PayloadOnline:     "alive",
		PayloadOffline:    "offline",
		DeviceClass:       "connectivity",
		AvailabilityTopic: config.MQTTTopic + "/availability",
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
}
