<a href="https://www.buymeacoffee.com/crmne" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

# MQTT Alive Daemon

MQTT Alive Daemon reports the status of your computer and custom commands to Home Assistant via MQTT. It's designed to be lightweight, easy to configure, and deployable across multiple machines.

## Features

- **Aliveness Reporting**: Regularly reports if the computer is online and running the daemon.
- **Custom Command Monitoring**: Execute and report the status of user-defined commands.
- **Home Assistant Integration**: Uses MQTT discovery for seamless integration with Home Assistant.
- **Multi-Machine Deployment**: Automatically generates a unique client ID for each machine, allowing easy deployment across multiple computers.
- **Flexible Configuration**: Simple YAML configuration file for easy setup and modification.

## Installation

### Prerequisites

- Go 1.16 or later
- Git
- Root access (sudo)

### Installation Steps

1. Clone the repository:
   ```
   git clone https://github.com/crmne/mqtt-alive-daemon.git
   cd mqtt-alive-daemon
   ```

2. Build and install the application:
   ```
   make install
   ```

   This command will:
   - Build the application
   - Install the binary to `/usr/local/bin`
   - Copy an example configuration file to the appropriate location
   - Set up and start the system service (launchd on macOS, systemd on Linux)

3. Edit the configuration file:
   - On macOS: `/usr/local/etc/mqtt-alive-daemon/config.yaml`
   - On Linux: `/etc/mqtt-alive-daemon/config.yaml`

### Uninstallation

To uninstall the application and remove all associated files:

```
sudo make uninstall
```

## Configuration

The daemon looks for the configuration and device files in the following locations (in order):

1. `/etc/mqtt-alive-daemon/`
2. `/usr/local/etc/mqtt-alive-daemon/`
3. `~/.config/mqtt-alive-daemon/`
4. `~/Library/Application Support/mqtt-alive-daemon/` (macOS only)

The main configuration file is named `config.yaml`, and the device-specific configuration is stored in `device_config.json`.

Edit the `config.yaml` file:

```yaml
mqtt_broker: "mqtt://your-mqtt-broker:1883"
mqtt_username: "your_username"
mqtt_password: "your_password"
device_name: "My Computer"
interval: 10
commands:
  usb_audio:
    command: "lsusb | grep 'Audio Device'"
    device_class: "plug"
  disk_space:
    command: "df -h / | awk 'NR==2 {print $5}' | sed 's/%//' | awk '$1 < 90 {exit 1}'"
    device_class: "problem"
```

The `device_config.json` file is automatically generated and managed by the application. It stores a unique client ID for each machine, allowing for multi-machine deployment.

## Usage

After installation and configuration, the daemon will start automatically on system boot. You can manually start, stop, or check the status of the service:

- On macOS:
  ```
  sudo launchctl load /Library/LaunchDaemons/me.paolino.mqtt-alive-daemon.plist
  sudo launchctl unload /Library/LaunchDaemons/me.paolino.mqtt-alive-daemon.plist
  sudo launchctl list | grep mqtt-alive-daemon
  ```

- On Linux:
  ```
  sudo systemctl start mqtt-alive-daemon
  sudo systemctl stop mqtt-alive-daemon
  sudo systemctl status mqtt-alive-daemon
  ```

## Development

To build the application without installing:

```
make build
```

To run tests:

```
make test
```

To clean up build artifacts:

```
make clean
```

## Home Assistant Integration

The daemon will automatically create binary sensors in Home Assistant for the aliveness check and each configured command. You can use these sensors in automations, scripts, or display them on your dashboard.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License.
