# Packaging mqtt-alive-daemon

This repository is the upstream source of truth for release artifacts and shared
packaging assets. Distro-specific package recipes live in the package repository
for that distro (e.g. the AUR git repos), not in this repository.

## Upstream Release Assets

Releases are tagged `v<version>` (e.g. `v0.4.0`). Each tag publishes via
goreleaser:

- `mqtt-alive-daemon_<version>_linux_amd64.tar.gz`
- `mqtt-alive-daemon_<version>_linux_arm64.tar.gz`
- `mqtt-alive-daemon_<version>_darwin_amd64.tar.gz`
- `mqtt-alive-daemon_<version>_darwin_arm64.tar.gz`
- `mqtt-alive-daemon_<version>_windows_amd64.zip`
- `mqtt-alive-daemon_<version>_windows_arm64.zip`
- `checksums.txt`
- GitHub's automatic source archive for the tag

The linux binaries are built with cgo so `.local` (mDNS) broker names resolve
through glibc NSS; they require glibc 2.35+ (Ubuntu 22.04, Debian 12, Fedora 36
or newer). The darwin and windows binaries are pure Go and resolve hostnames
through their OS APIs.

The archives contain:

- `mqtt-alive-daemon` (or `mqtt-alive-daemon.exe`)
- `README.md`
- `LICENSE`
- `config.yaml.example`
- `packaging/systemd/mqtt-alive-daemon.service`
- `packaging/launchd/me.paolino.mqtt-alive-daemon.plist`

## Dependencies

Runtime:

- `bash` on Linux/macOS (command sensors run through `bash -c`), PowerShell on Windows
- `systemd` only for the packaged system service

Build time:

- Go matching the directive in `go.mod`

## Build From Source

Packagers should stamp the version through `pkg/mqttalive` and build with cgo
enabled, so hostname resolution goes through glibc NSS and mDNS (`.local`)
broker names resolve:

```sh
version=0.4.1
CGO_ENABLED=1 go build -trimpath \
  -ldflags "-s -w -X github.com/crmne/mqtt-alive-daemon/pkg/mqttalive.Version=${version}" \
  -o mqtt-alive-daemon .
```

## Installed Files

```text
/usr/bin/mqtt-alive-daemon
/etc/mqtt-alive-daemon/config.yaml.example
/usr/share/licenses/mqtt-alive-daemon/LICENSE
/usr/share/doc/mqtt-alive-daemon/README.md
```

For systemd-based distros, also install:

```text
/usr/lib/systemd/system/mqtt-alive-daemon.service
```

Do not enable or start the service from package scripts, and do not install a
live `config.yaml`, which holds MQTT credentials. Users opt in with:

```sh
install -m 600 /etc/mqtt-alive-daemon/config.yaml.example /etc/mqtt-alive-daemon/config.yaml
# edit config.yaml, then:
systemctl enable --now mqtt-alive-daemon
```

## AUR

The AUR packages are maintained in separate git repos, checked out locally under
`.tmp/` (gitignored):

- `.tmp/aur-mqtt-alive-daemon`: stable, builds from the release tag
  (`ssh://aur@aur.archlinux.org/mqtt-alive-daemon.git`)
- `.tmp/aur-mqtt-alive-daemon-git`: builds from the latest git commit
  (`ssh://aur@aur.archlinux.org/mqtt-alive-daemon-git.git`)

Release flow for the stable package:

```sh
cd .tmp/aur-mqtt-alive-daemon
# bump pkgver (and reset pkgrel=1) in PKGBUILD, then:
updpkgsums
makepkg --printsrcinfo > .SRCINFO
makepkg -f   # smoke test
git commit -am "Update to <version>"
git push origin master
```

The `-git` package only needs a push when the PKGBUILD itself changes.

## Package Status

Current status as of 2026-07-14:

| Channel | Status | Notes |
|---|---|---|
| Arch AUR | Published | Stable [`mqtt-alive-daemon`](https://aur.archlinux.org/packages/mqtt-alive-daemon) (source build) and VCS [`mqtt-alive-daemon-git`](https://aur.archlinux.org/packages/mqtt-alive-daemon-git), both built with cgo. |

## Smoke Tests

After packaging, run:

```sh
mqtt-alive-daemon & sleep 2; kill %1   # logs "Starting MQTT Alive Daemon v<version>"
test -f /etc/mqtt-alive-daemon/config.yaml.example
systemctl cat mqtt-alive-daemon
```
