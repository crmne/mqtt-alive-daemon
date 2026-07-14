package mqttalive

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/denisbrodbeck/machineid"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"gopkg.in/yaml.v2"
)

var Version = "0.4.0"

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
	defaultInterval = 10
)

func getConfigLocations() []string {
	locations := []string{}
	switch runtime.GOOS {
	case "windows":
		if programData := os.Getenv("PROGRAMDATA"); programData != "" {
			locations = append(locations, filepath.Join(programData, "mqtt-alive-daemon"))
		}
		if userConfigDir, err := os.UserConfigDir(); err == nil && userConfigDir != "" {
			locations = appendUniqueLocation(locations, filepath.Join(userConfigDir, "mqtt-alive-daemon"))
		}
	case "darwin":
		locations = append(locations, "/etc/mqtt-alive-daemon", "/usr/local/etc/mqtt-alive-daemon")
		if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
			locations = appendUniqueLocation(locations, filepath.Join(homeDir, ".config", "mqtt-alive-daemon"))
			locations = appendUniqueLocation(locations, filepath.Join(homeDir, "Library", "Application Support", "mqtt-alive-daemon"))
		}
	default:
		locations = append(locations, "/etc/mqtt-alive-daemon", "/usr/local/etc/mqtt-alive-daemon")
		if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
			locations = appendUniqueLocation(locations, filepath.Join(homeDir, ".config", "mqtt-alive-daemon"))
		}
	}

	return locations
}

func appendUniqueLocation(locations []string, location string) []string {
	for _, existing := range locations {
		if filepath.Clean(existing) == filepath.Clean(location) {
			return locations
		}
	}
	return append(locations, location)
}

func readConfig() (Config, error) {
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
		return Config{}, fmt.Errorf("could not find a valid configuration file")
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, err
	}

	if config.Interval <= 0 {
		config.Interval = defaultInterval
	}

	return config, nil
}

func getOrCreateDeviceConfig() (DeviceConfig, error) {
	configLocations := getConfigLocations()
	var deviceConfig DeviceConfig

	for _, dir := range configLocations {
		configPath := filepath.Join(dir, "device_config.json")
		data, err := os.ReadFile(configPath)
		if err == nil {
			err = json.Unmarshal(data, &deviceConfig)
			if err == nil && deviceConfig.ClientID != "" {
				return deviceConfig, nil
			}
		}
	}

	// If not found, generate new client ID
	id, err := machineid.ProtectedID("mqtt-alive-daemon")
	if err != nil {
		return DeviceConfig{}, fmt.Errorf("failed to generate machine ID: %w", err)
	}
	deviceConfig.ClientID = id[:32]

	for _, dir := range configLocations {
		if err := saveDeviceConfig(deviceConfig, dir); err == nil {
			return deviceConfig, nil
		}
	}
	return DeviceConfig{}, fmt.Errorf("failed to save device config in any location")
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

	var err error
	config, err = readConfig()
	if err != nil {
		return err
	}

	deviceConfig, err = getOrCreateDeviceConfig()
	if err != nil {
		return err
	}

	// Create MQTT client options
	opts := mqtt.NewClientOptions().
		AddBroker(config.MQTTBroker).
		SetClientID(deviceConfig.ClientID).
		SetUsername(config.MQTTUsername).
		SetPassword(config.MQTTPassword).
		SetWill(availabilityTopic(), "offline", 1, true).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetOnConnectHandler(onConnect)

	// Create MQTT client
	client = mqtt.NewClient(opts)

	// Connect to the MQTT broker
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	log.Println("Connected to MQTT broker:", config.MQTTBroker)

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, getSignals()...)

	// Start the main loop
	go runMainLoop()

	// Wait for a signal
	sig := <-signalChan
	log.Printf("Received signal: %v\n", sig)
	client.Publish(availabilityTopic(), 1, true, "offline").Wait()
	client.Disconnect(250)
	return nil
}

func onConnect(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
	publishDiscovery()
	client.Publish(availabilityTopic(), 1, true, "online")
}

func runMainLoop() {
	for {
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

func availabilityTopic() string {
	return fmt.Sprintf("%s/binary_sensor/%s/availability", discoveryPrefix, deviceConfig.ClientID)
}

func stateTopic(name string) string {
	return fmt.Sprintf("%s/binary_sensor/%s_%s/state", discoveryPrefix, deviceConfig.ClientID, name)
}

func publishState(name, state string) {
	topic := stateTopic(name)
	token := client.Publish(topic, 0, false, state)
	token.Wait()
	if err := token.Error(); err != nil {
		log.Printf("Failed to publish state to topic %s: %v", topic, err)
		return
	}
	log.Printf("Published state: %s to topic: %s\n", state, topic)
}

func runCommand(command string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}
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
		StateTopic:        stateTopic(name),
		PayloadOn:         "ON",
		PayloadOff:        "OFF",
		DeviceClass:       deviceClass,
		AvailabilityTopic: availabilityTopic(),
		Device: Device{
			Identifiers:  []string{deviceConfig.ClientID},
			Name:         config.DeviceName,
			Manufacturer: "MQTT Alive Daemon",
			Model:        fmt.Sprintf("v%s (%s/%s)", Version, runtime.GOOS, runtime.GOARCH),
			SwVersion:    Version,
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
	if err := token.Error(); err != nil {
		log.Printf("Failed to publish discovery message for %s: %v", name, err)
		return
	}

	log.Printf("Published discovery message for %s to topic: %s\n", name, discoveryTopic)
}
