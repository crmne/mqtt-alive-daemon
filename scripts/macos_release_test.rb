require 'minitest/autorun'
require 'yaml'
require_relative 'macos_release'

class MacosReleaseTest < Minitest::Test
  def setup
    @directory = Dir.mktmpdir('mqtt-signing-test-')
    @root = File.join(@directory, 'source')
    @signed = File.join(@directory, 'signed')
    @destination = File.join(@directory, 'binary')
    File.write(@destination, 'original')
    MacosRelease::ASSETS.each do |path|
      FileUtils.mkdir_p(File.dirname(File.join(@root, path)))
      File.write(File.join(@root, path), path)
    end
    MacosRelease::ARCHITECTURES.each do |arch, cpu|
      FileUtils.mkdir_p(File.join(@signed, arch))
      File.binwrite(File.join(@signed, arch, MacosRelease::BINARY), [0xfeedfacf, cpu].pack('L<L<') + 'fixture')
      MacosRelease::ASSETS.each do |path|
        target = File.join(@signed, arch, path)
        FileUtils.mkdir_p(File.dirname(target))
        FileUtils.cp(File.join(@root, path), target)
      end
    end
    @manifest = {
      'schema' => 1, 'tool' => 'native-packages 0.5.0', 'version' => '1.2.3', 'commit' => 'abc123',
      'files' => MacosRelease.payload_files.to_h { |p| [p, Digest::SHA256.file(File.join(@signed, p)).hexdigest] }
    }
    save_manifest
    @environment = { 'MQTT_SIGNED_DARWIN' => @signed, 'MQTT_REQUIRE_SIGNED_DARWIN' => '1' }
  end

  def teardown
    FileUtils.remove_entry(@directory)
  end

  def save_manifest
    File.write(File.join(@signed, 'manifest.json'), JSON.generate(@manifest))
  end

  def import(os = 'darwin', arch = 'amd64', version = '1.2.3', commit = 'abc123')
    MacosRelease.import(os, arch, @destination, version, commit, environment: @environment, root: @root)
  end

  def test_imports_exact_bytes_and_restores_artifact_executable_permission
    File.chmod(0o644, File.join(@signed, 'amd64', MacosRelease::BINARY))
    import
    assert_equal File.binread(File.join(@signed, 'amd64', MacosRelease::BINARY)), File.binread(@destination)
    assert_equal 0o755, File.stat(@destination).mode & 0o777
  end

  def test_other_platforms_are_untouched_even_without_signing_inputs
    @environment.clear
    %w[windows linux].each { |os| import(os) }
    assert_equal 'original', File.read(@destination)
  end

  def test_release_cannot_fall_back_to_unsigned_binaries
    @environment.delete('MQTT_SIGNED_DARWIN')
    assert_raises(RuntimeError) { import }
    assert_equal 'original', File.read(@destination)
  end

  def test_rejects_other_versions_and_commits
    assert_raises(RuntimeError) { import('darwin', 'amd64', '1.2.4') }
    assert_raises(RuntimeError) { import('darwin', 'amd64', '1.2.3', 'other') }
    assert_equal 'original', File.read(@destination)
  end

  def test_rejects_changed_or_missing_signed_files
    File.write(File.join(@signed, 'arm64', MacosRelease::BINARY), 'tampered')
    assert_raises(RuntimeError) { import }
    @manifest['files'].delete('arm64/' + MacosRelease::BINARY)
    save_manifest
    assert_raises(RuntimeError) { import }
    assert_equal 'original', File.read(@destination)
  end

  def test_rejects_architecture_swaps_even_with_updated_hashes
    path = File.join(@signed, 'amd64', MacosRelease::BINARY)
    FileUtils.cp(File.join(@signed, 'arm64', MacosRelease::BINARY), path)
    @manifest['files']['amd64/' + MacosRelease::BINARY] = Digest::SHA256.file(path).hexdigest
    save_manifest
    assert_raises(RuntimeError) { import }
    assert_equal 'original', File.read(@destination)
  end

  def test_rejects_archive_asset_changes
    File.write(File.join(@root, 'config.yaml.example'), 'changed')
    assert_raises(RuntimeError) { import }
    assert_equal 'original', File.read(@destination)
  end

  def test_staged_assets_match_the_goreleaser_archive
    config = YAML.safe_load_file(File.expand_path('../.goreleaser.yml', __dir__))
    assert_equal config.fetch('archives').first.fetch('files').sort, MacosRelease::ASSETS.sort
  end
end
