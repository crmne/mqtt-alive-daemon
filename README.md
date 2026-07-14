<div align="center">

# MQTT Alive Daemon

**Turn any computer into a Home Assistant device.**

[![GitHub Release](https://img.shields.io/github/v/release/crmne/mqtt-alive-daemon)](https://github.com/crmne/mqtt-alive-daemon/releases)
[![AUR](https://img.shields.io/aur/version/mqtt-alive-daemon)](https://aur.archlinux.org/packages/mqtt-alive-daemon)
[![CI](https://github.com/crmne/mqtt-alive-daemon/actions/workflows/ci.yml/badge.svg)](https://github.com/crmne/mqtt-alive-daemon/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

</div>

---

mqtt-alive-daemon is a small daemon that reports whether your machines are alive, and anything a shell command can check, as binary sensors in Home Assistant over MQTT with automatic discovery. Install it on every computer you own and each one shows up as its own device.

## What you get

- **Aliveness sensor**: each machine reports online/offline, with an MQTT last will so crashes and shutdowns show up too
- **Command sensors**: any shell command becomes a binary sensor; exit 0 turns it ON
- **Automatic discovery**: sensors appear in Home Assistant by themselves, no YAML on the HA side
- **One device per machine**: a stable client ID derived from the machine ID, so the same config deploys everywhere
- **Cross-platform**: Linux, macOS, and Windows; commands run through bash or PowerShell
- **One binary and a YAML file**: no runtime dependencies beyond a shell

## Install

Arch Linux:

```bash
yay -S mqtt-alive-daemon
# or
yay -S mqtt-alive-daemon-git
```

macOS and Linux, via Homebrew:

```bash
brew install crmne/tap/mqtt-alive-daemon
brew services start mqtt-alive-daemon
```

Prebuilt binaries for Linux, macOS, and Windows are on the [releases page](https://github.com/crmne/mqtt-alive-daemon/releases). The Linux binaries need glibc 2.35+ (Ubuntu 22.04, Debian 12, Fedora 36 or newer).

macOS and Linux, from source:

```bash
git clone https://github.com/crmne/mqtt-alive-daemon.git
cd mqtt-alive-daemon
make install
```

This builds the binary, installs it to `/usr/local/bin`, copies an example config into place, and sets up the system service (launchd on macOS, systemd on Linux).

Windows, from source (defaults to per-user in `%APPDATA%`):

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1
# or for all users (requires admin):
powershell -ExecutionPolicy Bypass -File .\scripts\install.ps1 -InstallScope AllUsers
```

This registers a Scheduled Task that runs at logon (`CurrentUser`) or system startup (`AllUsers`), falling back to a Startup folder entry if Scheduled Tasks are blocked.

Distro packagers should use [PACKAGING.md](PACKAGING.md).

## Configure

Edit `config.yaml`:

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

Every `interval` seconds the daemon runs each command; exit 0 turns the sensor ON. `device_class` is any [Home Assistant binary sensor class](https://www.home-assistant.io/integrations/binary_sensor/#device-class) and defaults to `problem`.

The daemon looks for `config.yaml` in these locations, in order:

1. `/etc/mqtt-alive-daemon/`
2. `/usr/local/etc/mqtt-alive-daemon/`
3. `%ProgramData%\mqtt-alive-daemon\` (Windows only)
4. `~/.config/mqtt-alive-daemon/`
5. `~/Library/Application Support/mqtt-alive-daemon/` (macOS only)
6. `%APPDATA%\mqtt-alive-daemon\` (Windows only)

On Windows, commands run through PowerShell:

```yaml
commands:
  mg_xu:
    command: "Get-PnpDevice -PresentOnly | Where-Object { $_.FriendlyName -like '*MG-XU*' -and $_.Status -eq 'OK' } | Select-Object -First 1 | ForEach-Object { 'OK' }"
    device_class: "plug"
```

A `device_config.json` appears next to the config on first run. It stores the machine's client ID. Leave it alone: it keeps the Home Assistant device identity stable across upgrades.

## Enable the service

Package and `make install` setups start on boot already. To manage the service:

Linux:

```bash
sudo systemctl enable --now mqtt-alive-daemon
sudo systemctl status mqtt-alive-daemon
```

macOS:

```bash
sudo launchctl load /Library/LaunchDaemons/me.paolino.mqtt-alive-daemon.plist
sudo launchctl list | grep mqtt-alive-daemon
```

The sensors appear in Home Assistant under **Settings → Devices & Services → MQTT** as soon as the daemon connects.

## Use cases

- **USB devices**: is the audio interface actually plugged in?
  `lsusb | grep 'Audio Device'` (Linux), `ioreg -p IOUSB -l -w 0 | grep 'Device Name'` (macOS)
- **Disk space**: alert before a disk fills up
  `df -h / | awk 'NR==2 {print $5}' | sed 's/%//' | awk '$1 < 90 {exit 1}'`
- **Processes**: is the backup agent running?
  `pgrep -x borg`
- **Network**: can this machine reach a host?
  `ping -c 1 example.com`
- **Anything else**: if a shell one-liner can check it, it can be a sensor in Home Assistant

## Why it exists

Home Assistant can tell you about your lights, but not whether the studio Mac still sees its audio interface, whether the office PC's disk is filling up, or whether a machine is even switched on. Anything you can check from a shell should be a sensor, without writing an integration and without YAML on the Home Assistant side.

## How it works

On connect, the daemon publishes retained [MQTT discovery](https://www.home-assistant.io/integrations/mqtt/#mqtt-discovery) configs, so Home Assistant creates the device and its sensors automatically. Every `interval` seconds it publishes the aliveness sensor and one state per command. An MQTT last will marks the device offline the moment the connection drops, and the client ID, derived from the machine ID and persisted in `device_config.json`, keeps each machine's identity stable across reinstalls and upgrades.

## Uninstall

```bash
sudo make uninstall   # keeps /etc/mqtt-alive-daemon (credentials and device identity)
sudo make purge       # removes it too
```

On Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\uninstall.ps1                # keeps config
powershell -ExecutionPolicy Bypass -File .\scripts\uninstall.ps1 -RemoveConfig # removes it too
```

## Development

```bash
make build   # build without installing
make test    # run tests
make clean   # remove build artifacts
```

On Windows, `scripts\build.ps1` and `scripts\run.ps1` do the same.

## Support

If this saves you a trip to the server closet, you can [buy me a coffee](https://www.buymeacoffee.com/crmne).

## License

MIT
