<a href="https://www.buymeacoffee.com/crmne" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

# MQTT Alive Daemon

MQTT Alive Daemon reports the status of your computer and custom commands to Home Assistant via MQTT. It's designed to be lightweight, easy to configure, and deployable across multiple machines.

## Features

- **Aliveness Reporting**: Regularly reports if the computer is online and running the daemon.
- **Custom Command Monitoring**: Execute and report the status of user-defined commands.
- **Home Assistant Integration**: Uses MQTT discovery for seamless integration with Home Assistant.
- **Multi-Machine Deployment**: Automatically generates a unique client ID for each machine, allowing easy deployment across multiple computers.
- **Flexible Configuration**: Simple YAML configuration file for easy setup and modification.

## Use Cases

With MQTT Alive Daemon, you can monitor various aspects of your computer(s) in Home Assistant, such as:

1. **USB Device Connection**: Check if specific USB devices are connected.  
   Example command: `lsusb | grep "Device Name"` (Linux)  
   Example command: `ioreg -p IOUSB -l -w 0 | grep "Device Name"` (macOS)

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

### Prerequisites

- Go 1.16 or later
- Git
- Root access (sudo) on macOS/Linux
  - On Windows, administrator access is only required if you want to write to `%ProgramData%`.

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
   - On Windows: `%ProgramData%\mqtt-alive-daemon\config.yaml` or `%APPDATA%\mqtt-alive-daemon\config.yaml`

### Windows (PowerShell)

1. Build the application:
   ```
   powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1
   ```
2. Install (defaults to per-user in `%APPDATA%`):
   ```
   powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1
   ```
   To install for all users (requires admin):
   ```
   powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -InstallScope AllUsers
   ```
   This also registers a Windows Scheduled Task named `mqtt-alive-daemon`:
   - `AllUsers`: runs at system startup as `SYSTEM`
   - `CurrentUser`: runs at user logon
   If Scheduled Tasks are blocked, it falls back to a Startup folder entry.
3. Edit the configuration file:
   - `%APPDATA%\mqtt-alive-daemon\config.yaml`
   - Or `%ProgramData%\mqtt-alive-daemon\config.yaml` if you install with `-InstallScope AllUsers`
4. Run the daemon:
   ```
   powershell -ExecutionPolicy Bypass -File .\scripts\run.ps1
   ```

### Uninstallation

To uninstall the application and remove all associated files:

```
sudo make uninstall
```

On Windows:

```
powershell -ExecutionPolicy Bypass -File .\scripts\uninstall.ps1
```
## Configuration

The daemon looks for the configuration and device files in the following locations (in order):

1. `/etc/mqtt-alive-daemon/`
2. `/usr/local/etc/mqtt-alive-daemon/`
3. `%ProgramData%\mqtt-alive-daemon\` (Windows only)
4. `~/.config/mqtt-alive-daemon/`
5. `~/Library/Application Support/mqtt-alive-daemon/` (macOS only)
6. `%APPDATA%\mqtt-alive-daemon\` (Windows only)

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

On Windows, commands are executed via PowerShell, so adjust the command syntax accordingly.

Windows examples:

```yaml
commands:
  mg_xu:
    command: "Get-PnpDevice -PresentOnly | Where-Object { $_.FriendlyName -like '*MG-XU*' -and $_.Status -eq 'OK' } | Select-Object -First 1 | ForEach-Object { 'OK' }"
    device_class: "plug"
  usb_2_5g_lan:
    command: "Get-NetAdapter | Where-Object { $_.InterfaceDescription -like '*Realtek*USB*2.5GbE*' -and $_.Status -eq 'Up' } | Select-Object -First 1 | ForEach-Object { 'OK' }"
    device_class: "plug"
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

On Windows:

```
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1
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
