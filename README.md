# MQTT Alive Daemon

MQTT Alive Daemon is a versatile Go application that reports the status of your computer and custom commands to Home Assistant via MQTT. It's designed to be lightweight, easy to configure, and deployable across multiple machines.

## Features

- **Aliveness Reporting**: Regularly reports if the computer is online and running the daemon.
- **Custom Command Monitoring**: Execute and report the status of user-defined commands.
- **Home Assistant Integration**: Uses MQTT discovery for seamless integration with Home Assistant.
- **Multi-Machine Deployment**: Automatically generates a unique client ID for each machine, allowing easy deployment across multiple computers.
- **Flexible Configuration**: Simple YAML configuration file for easy setup and modification.

## Use Cases

With MQTT Alive Daemon, you can monitor various aspects of your computer(s) in Home Assistant, such as:

1. **USB Device Connection**: Check if specific USB devices are connected.
   Example command: `lsusb | grep "Device Name"`

2. **Disk Space**: Monitor available disk space.
   Example command: `df -h / | awk 'NR==2 {print $5}' | sed 's/%//' | awk '$1 < 90 {exit 1}'`

3. **Process Running**: Check if a particular process is running.
   Example command: `pgrep -x "process_name" > /dev/null && echo "Running" || echo "Not running"`

4. **Network Connectivity**: Test connection to a specific host.
   Example command: `ping -c 1 example.com > /dev/null && echo "Reachable" || echo "Unreachable"`

5. **Temperature Monitoring**: Report CPU temperature (on supported systems).
   Example command: `sensors | grep "CPU Temperature" | awk '{print $3}' | cut -c2-3`

6. **Battery Status**: Check laptop battery level (on supported systems).
   Example command: `pmset -g batt | grep -Eo "\d+%" | cut -d% -f1`

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/crmne/mqtt-alive-daemon.git
   ```

2. Build the application:
   ```
   cd mqtt-alive-daemon
   go build -o mqtt-alive-daemon
   ```

## Configuration

1. Create the configuration directory:
   ```
   mkdir -p ~/.config/mqtt-alive-daemon
   ```

2. Create a `config.yaml` file in the configuration directory:
   ```
   nano ~/.config/mqtt-alive-daemon/config.yaml
   ```

3. Add your configuration to the file:

   ```yaml
   mqtt_broker: "mqtt://your-mqtt-broker:1883"
   mqtt_username: "your_username"
   mqtt_password: "your_password"
   device_name: "My Computer"
   interval: 10
   commands:
     usb_audio:
       command: "lsusb | grep 'Audio Device' > /dev/null && echo 'Connected' || echo 'Disconnected'"
       device_class: "plug"
     disk_space:
       command: "df -h / | awk 'NR==2 {print $5}' | sed 's/%//' | awk '$1 < 90 {exit 1}'"
       device_class: "plug"
   ```

4. Secure the configuration file:
   ```
    chmod 600 ~/.config/mqtt-alive-daemon/config.yaml
   ```

## Running the Daemon

Run the daemon:

```
./mqtt-alive-daemon
```

For automatic startup, you can create a systemd service (Linux) or launchd job (macOS).

## Home Assistant Integration

The daemon will automatically create binary sensors in Home Assistant for the aliveness check and each configured command. You can use these sensors in automations, scripts, or display them on your dashboard.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
