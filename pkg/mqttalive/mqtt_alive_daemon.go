package mqttalive

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/denisbrodbeck/machineid"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"
)

var Version = "0.2.0"

type Config struct {
	MQTTBroker   string                   `yaml:"mqtt_broker"`
	MQTTUsername string                   `yaml:"mqtt_username"`
	MQTTPassword string                   `yaml:"mqtt_password"`
	DeviceName   string                   `yaml:"device_name"`
	Commands     map[string]CommandConfig `yaml:"commands"`
	Interval     int                      `yaml:"interval"`
}

type CommandConfig struct {
	Command     string `yaml:"command"`
	DeviceClass string `yaml:"device_class"`
}

type DeviceConfig struct {
	ClientID string `json:"client_id"`
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
	SwVersion    string   `json:"sw_version"`
}

var client mqtt.Client
var config Config
var deviceConfig DeviceConfig

const (
	discoveryPrefix = "homeassistant"
)

var (
	systemConfigDirs = []string{
		"/etc/mqtt-alive-daemon",
		"/usr/local/etc/mqtt-alive-daemon",
	}
	userConfigDirs = []string{
		filepath.Join(os.Getenv("HOME"), ".config/mqtt-alive-daemon"),
		filepath.Join(os.Getenv("HOME"), "Library/Application Support/mqtt-alive-daemon"),
	}
)

func getConfigLocations() []string {
	return append(systemConfigDirs, userConfigDirs...)
}

func readConfig() Config {
	configLocations := getConfigLocations()

	var config Config
	var err error
	var data []byte

	for _, location := range configLocations {
		data, err = os.ReadFile(filepath.Join(location, "config.yaml"))
		if err == nil {
			break
		}
	}

	if err != nil {
		log.Fatal("Could not find a valid configuration file")
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func getOrCreateDeviceConfig() DeviceConfig {
	configLocations := getConfigLocations()
	var deviceConfig DeviceConfig

	for _, dir := range configLocations {
		configPath := filepath.Join(dir, "device_config.json")
		data, err := os.ReadFile(configPath)
		if err == nil {
			err = json.Unmarshal(data, &deviceConfig)
			if err == nil && deviceConfig.ClientID != "" {
				return deviceConfig
			}
		}
	}

	// If not found, generate new client ID
	id, err := machineid.ProtectedID("mqtt-alive-daemon")
	if err != nil {
		log.Fatal("Failed to generate machine ID:", err)
	}
	hash := sha256.Sum256([]byte(id))
	deviceConfig.ClientID = hex.EncodeToString(hash[:])[:32]

	for _, dir := range configLocations {
		if err := saveDeviceConfig(deviceConfig, dir); err == nil {
			return deviceConfig
		}
	}
	log.Fatal("Failed to save device config in any location")
	return DeviceConfig{} // This line will never be reached, but it's needed for compilation
}

func saveDeviceConfig(deviceConfig DeviceConfig, dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "device_config.json")
	data, err := json.Marshal(deviceConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func Run() error {
	log.Printf("Starting MQTT Alive Daemon v%s\n", Version)

	// Read configuration
	config = readConfig()

	// Get or generate client ID
	deviceConfig = getOrCreateDeviceConfig()

	// Create MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(config.MQTTBroker).
		SetClientID(deviceConfig.ClientID).
		SetUsername(config.MQTTUsername).
		SetPassword(config.MQTTPassword).
		SetWill(fmt.Sprintf("%s/binary_sensor/%s/availability", discoveryPrefix, deviceConfig.ClientID), "offline", 1, true).
		SetAutoReconnect(true).
		SetOnConnectHandler(onConnect)

	// Create MQTT client
	client = mqtt.NewClient(opts)

	// Connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	log.Println("Connected to MQTT broker:", config.MQTTBroker)

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the main loop
	go runMainLoop()

	// Wait for signals
	for {
		select {
		case sig := <-signalChan:
			log.Printf("Received signal: %v\n", sig)
			client.Publish(fmt.Sprintf("%s/binary_sensor/%s/availability", discoveryPrefix, deviceConfig.ClientID), 1, true, "offline")
			client.Disconnect(250)
			return nil
		}
	}
}

func onConnect(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
	publishDiscovery()
	client.Publish(fmt.Sprintf("%s/binary_sensor/%s/availability", discoveryPrefix, deviceConfig.ClientID), 1, true, "online")
}

func runMainLoop() {
	for {
		checkMQTTConnection()
		publishState("aliveness", "ON")
		for name, command := range config.Commands {
			state := "OFF"
			if err := runCommand(command.Command); err == nil {
				state = "ON"
			}
			publishState(name, state)
		}
		time.Sleep(time.Duration(config.Interval) * time.Second)
	}
}

func checkMQTTConnection() {
	if !client.IsConnected() {
		log.Println("MQTT connection lost. Attempting to reconnect...")
		if token := client.Connect(); token.Wait() && token.Error() != nil {
			log.Printf("Failed to reconnect to MQTT broker: %v", token.Error())
			return
		}
		log.Println("Reconnected to MQTT broker")
	}
}

func publishState(name, state string) {
	topic := fmt.Sprintf("%s/binary_sensor/%s_%s/state", discoveryPrefix, deviceConfig.ClientID, name)
	token := client.Publish(topic, 0, false, state)
	token.Wait()
	log.Printf("Published state: %s to topic: %s\n", state, topic)
}

func runCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	return cmd.Run()
}

func publishDiscovery() {
	publishSensorDiscovery("aliveness", "Aliveness", "connectivity")
	for name, cmdConfig := range config.Commands {
		deviceClass := cmdConfig.DeviceClass
		if deviceClass == "" {
			deviceClass = "problem" // Default to "problem" if not specified
		}
		publishSensorDiscovery(name, name, deviceClass)
	}
}

func publishSensorDiscovery(name, displayName, deviceClass string) {
	payload := DiscoveryPayload{
		Name:              displayName,
		UniqueID:          fmt.Sprintf("%s_%s", deviceConfig.ClientID, name),
		StateTopic:        fmt.Sprintf("%s/binary_sensor/%s_%s/state", discoveryPrefix, deviceConfig.ClientID, name),
		PayloadOn:         "ON",
		PayloadOff:        "OFF",
		DeviceClass:       deviceClass,
		AvailabilityTopic: fmt.Sprintf("%s/binary_sensor/%s/availability", discoveryPrefix, deviceConfig.ClientID),
		Device: Device{
			Identifiers:  []string{deviceConfig.ClientID},
			Name:         config.DeviceName,
			Manufacturer: "MQTT Alive Daemon",
			Model:        fmt.Sprintf("v%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH),
			SwVersion:    fmt.Sprintf("%s", Version),
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling discovery payload: %v", err)
		return
	}

	discoveryTopic := fmt.Sprintf("%s/binary_sensor/%s_%s/config", discoveryPrefix, deviceConfig.ClientID, name)
	token := client.Publish(discoveryTopic, 0, true, payloadJSON)
	token.Wait()

	log.Printf("Published discovery message for %s to topic: %s\n", name, discoveryTopic)
}
