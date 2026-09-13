# Packaging

[`native-packages.yaml`](native-packages.yaml) is the packaging configuration:
it pins the shared CLI and nFPM versions and declares Linux amd64/arm64 inputs,
DEB/RPM contents, dependencies, recipe templates and downstream repositories.
Application assets and native recipes stay in `packaging/`.

```sh
gem install native-packages --version 0.2.0
native-packages validate
native-packages doctor
native-packages build --release v1.2.3
```

Replace `v1.2.3` with an existing stable application release. Local use also
requires nFPM 2.47.0, `bsdtar` and `readelf`; AUR generation needs `makepkg`
or Docker. CI installs its tooling. To package local release archives, put
every configured input and recipe asset under `dist/`, then run
`native-packages build --version 1.2.3`. Outputs go to
`dist/packages/1.2.3`; use `--output` for a fresh destination when rebuilding.

Stable tags run the existing native build jobs first. After binaries and
`checksums.txt` are published, the shared workflow verifies their hashes,
builds the configured packages, and attaches them to the GitHub release.
Configured recipes are attached as an archive. Package checksums are separate
from the original binary checksums. PR validation never publishes.

Review or publish an existing build with the same installed CLI:

```sh
native-packages publish --from dist/packages/1.2.3 --to github
native-packages repositories
native-packages status --offline
```

For applications with configured AUR or Homebrew destinations, stage the
recipes with `native-packages stage TARGET dist/packages/1.2.3/recipes`,
inspect `native-packages diff TARGET`, run native package validation, and
publish with `native-packages publish TARGET`. These destinations use ignored
managed Git clones, recorded in this application's YAML configuration.
AUR automation needs `PUBLISH_AUR=true`, `AUR_SSH_KEY` and `AUR_KNOWN_HOSTS`;
Homebrew automation needs `PUBLISH_HOMEBREW=true` and
`HOMEBREW_TAP_GITHUB_TOKEN`. Enable only configured destinations.

The existing macOS, Windows and Flatpak build/signing steps remain responsible
for their native artifacts. Additional nFPM formats require suitable platform
inputs and dependencies; adding a format does not port the application.
See the [shared CLI documentation](https://github.com/crmne/native-packages/tree/v0.2.0)
for commands and supported formats.

To upgrade the tool, change `tool.version` in `native-packages.yaml`, the
matching immutable workflow reference, and any release-job gem installation
pin together. Applications need no packaging Gemfile, lockfile or Ruby wrapper.

GoReleaser continues to build the native archives and publish the Homebrew
formula with its service integration. Do not enable a second Homebrew publisher.

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
- Linux DEB/RPM files and `packaging-checksums.txt` after stable packaging succeeds
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

The stable and Git recipes live in `packaging/arch/`. Their independent AUR Git
repositories are registered in `native-packages.yaml`; the shared CLI
creates managed checkouts under `.cache/packaging/repos/`.

```sh
native-packages build --release v0.4.1
native-packages stage aur dist/packages/0.4.1/recipes
native-packages diff aur
# Validate the native package builds before publishing.
native-packages publish aur
```

Automatic stable-release publication requires `PUBLISH_AUR=true`, `AUR_SSH_KEY`
and `AUR_KNOWN_HOSTS`. The existing `.tmp/` clones remain historical workspaces;
new recipe edits belong here. A Git recipe change does not require an app release.

## Homebrew

The formula lives in [`crmne/homebrew-tap`](https://github.com/crmne/homebrew-tap)
and is regenerated by goreleaser on every release, using the
`HOMEBREW_TAP_GITHUB_TOKEN` repository secret (a GitHub token with write access
to the tap). goreleaser deprecated `brews` in favor of `homebrew_casks`, but
casks have no `brew services` integration and don't work on Linux, so the
formula stays. If a future goreleaser drops `brews`, pin the
goreleaser-action `version` to one that still has it.

## Package Status

Current status as of 2026-07-14:

| Channel | Status | Notes |
|---|---|---|
| Arch AUR | Published | Stable [`mqtt-alive-daemon`](https://aur.archlinux.org/packages/mqtt-alive-daemon) (source build) and VCS [`mqtt-alive-daemon-git`](https://aur.archlinux.org/packages/mqtt-alive-daemon-git), both built with cgo. |
| Homebrew | Published | Formula in [`crmne/homebrew-tap`](https://github.com/crmne/homebrew-tap) with `brew services` support, auto-updated by goreleaser on each release. |

## Smoke Tests

After packaging, run:

```sh
mqtt-alive-daemon & sleep 2; kill %1   # logs "Starting MQTT Alive Daemon v<version>"
test -f /etc/mqtt-alive-daemon/config.yaml.example
systemctl cat mqtt-alive-daemon
```
