#!/usr/bin/env ruby
# GoReleaser owns compilation, archives, checksums and Homebrew. The published
# native-packages gem owns Apple signing; this bridges the two native hosts.
require 'digest'
require 'fileutils'
require 'json'
require 'open3'
require 'tmpdir'

module MacosRelease
  BINARY = 'mqtt-alive-daemon'
  ARCHITECTURES = { 'amd64' => 0x01000007, 'arm64' => 0x0100000c }.freeze
  ASSETS = %w[README.md LICENSE config.yaml.example
    packaging/systemd/mqtt-alive-daemon.service
    packaging/launchd/me.paolino.mqtt-alive-daemon.plist].freeze
  APPLE_KEYS = %w[APPLE_CERTIFICATE_P12 APPLE_CERTIFICATE_PASSWORD
    APPLE_SIGNING_IDENTITY APPLE_ID APPLE_TEAM_ID APPLE_APP_PASSWORD].freeze
  module_function

  def run(*command)
    raise "Command failed: #{command.first}" unless system(*command)
  end

  def check_architecture(path, arch)
    raise "Expected a regular binary: #{path}" unless File.lstat(path).file?
    header = File.binread(path, 8).unpack('L<L<')
    raise "Wrong Mach-O architecture for #{arch}" unless header == [0xfeedfacf, ARCHITECTURES.fetch(arch)]
  end

  def payload_files
    ARCHITECTURES.keys.flat_map { |arch| [BINARY, *ASSETS].map { |path| "#{arch}/#{path}" } }
  end

  def sign(dist, output)
    raise 'Signing requires a native Mac' unless RUBY_PLATFORM.include?('darwin')
    missing = APPLE_KEYS.select { |key| ENV[key].to_s.empty? }
    raise "Release signing requires: #{missing.join(', ')}" unless missing.empty?
    raise 'Signed output must be new' if File.exist?(output) || File.symlink?(output)
    metadata = JSON.parse(File.read(File.join(dist, 'metadata.json')))
    commit, status = Open3.capture2('git', 'rev-parse', 'HEAD')
    raise 'Build belongs to another checkout' unless status.success? && metadata.fetch('commit') == commit.strip
    artifacts = JSON.parse(File.read(File.join(dist, 'artifacts.json'))).select do |artifact|
      artifact['type'] == 'Binary' && artifact['goos'] == 'darwin' && artifact.dig('extra', 'ID') == 'portable'
    end
    raise 'Expected both Darwin architectures exactly once' unless artifacts.map { |a| a.fetch('goarch') }.sort == ARCHITECTURES.keys.sort
    Dir.mktmpdir('mqtt-darwin-input-') do |input|
      artifacts.each do |artifact|
        arch = artifact.fetch('goarch')
        binary = File.realpath(artifact.fetch('path'))
        raise 'Binary is outside the build output' unless binary.start_with?(File.realpath(dist) + File::SEPARATOR)
        check_architecture(binary, arch)
        directory = File.join(input, arch)
        FileUtils.mkdir_p(directory)
        FileUtils.cp(binary, File.join(directory, BINARY), preserve: true)
        ASSETS.each do |path|
          destination = File.join(directory, path)
          FileUtils.mkdir_p(File.dirname(destination))
          FileUtils.cp(path, destination, preserve: true)
        end
      end
      FileUtils.mkdir_p(File.dirname(File.expand_path(output)))
      run(Gem.ruby, '-e', 'gem "native-packages", "0.5.0"; load Gem.bin_path("native-packages", "native-packages", "0.5.0")',
        'notarize-macos', input, '--output', output)
    end
    ARCHITECTURES.each_key do |arch|
      binary = File.join(output, arch, BINARY)
      check_architecture(binary, arch)
      run('codesign', '--verify', '--strict', '-R=notarized', '--check-notarization', binary)
    end
    manifest = {
      'schema' => 1, 'tool' => 'native-packages 0.5.0',
      'version' => metadata.fetch('version'), 'commit' => metadata.fetch('commit'),
      'files' => payload_files.to_h { |path| [path, Digest::SHA256.file(File.join(output, path)).hexdigest] }
    }
    File.write(File.join(output, 'manifest.json'), JSON.pretty_generate(manifest) + "\n")
    puts 'Both Darwin payloads are notarized and ready for GoReleaser.'
  end

  def import(os, arch, destination, version, commit, environment: ENV, root: Dir.pwd)
    return unless os == 'darwin'
    signed = environment['MQTT_SIGNED_DARWIN']
    if signed.to_s.empty?
      raise 'Missing signed Darwin payloads' if environment['MQTT_REQUIRE_SIGNED_DARWIN'] == '1'
      puts 'Local build: Darwin signing is deferred to the native Mac release job.'
      return
    end
    raise "Unsupported Darwin architecture: #{arch}" unless ARCHITECTURES.key?(arch)
    manifest = JSON.parse(File.read(File.join(signed, 'manifest.json')))
    raise 'Unsupported signing manifest' unless manifest.fetch('schema') == 1 && manifest.fetch('tool') == 'native-packages 0.5.0'
    raise 'Signed payload version/commit does not match this build' unless manifest.fetch('version') == version && manifest.fetch('commit') == commit
    files = manifest.fetch('files')
    raise 'Incomplete signed payload manifest' unless files.keys.sort == payload_files.sort
    files.each do |path, digest|
      source = File.join(signed, path)
      raise "Invalid signed payload file: #{path}" unless File.lstat(source).file? && Digest::SHA256.file(source).hexdigest == digest
    end
    ASSETS.each do |path|
      raise "Archive asset changed since signing: #{path}" unless Digest::SHA256.file(File.join(root, path)).hexdigest == files.fetch("#{arch}/#{path}")
    end
    source = File.join(signed, arch, BINARY)
    check_architecture(source, arch)
    FileUtils.cp(source, destination)
    File.chmod(0o755, destination) # Actions artifacts do not preserve executable modes.
    raise 'Signed binary changed while copying' unless Digest::SHA256.file(destination).hexdigest == files.fetch("#{arch}/#{BINARY}")
    puts "Imported notarized Darwin #{arch} binary before archiving."
  end
end

if $PROGRAM_NAME == __FILE__
  begin
    case ARGV.shift
    when 'sign'
      raise 'Usage: macos_release.rb sign DIST FRESH_OUTPUT' unless ARGV.length == 2
      MacosRelease.sign(*ARGV)
    when 'import'
      raise 'Usage: macos_release.rb import OS ARCH BINARY VERSION COMMIT' unless ARGV.length == 5
      MacosRelease.import(*ARGV)
    else
      raise 'Expected sign or import'
    end
  rescue StandardError => error
    warn error.message
    exit 1
  end
end
