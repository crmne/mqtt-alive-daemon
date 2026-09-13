require 'json'
require 'tmpdir'
require 'fileutils'
require 'shellwords'
gem 'native-packages', '0.5.0'
require 'native_packages/build'

artifacts = JSON.parse(File.read('dist/artifacts.json')).select { |a| a['type'] == 'Binary' && a['goos'] == 'darwin' }
probe_directory = Dir.mktmpdir('mqtt-clang-diagnostic-')
probe = File.join(probe_directory, 'clang-cli')
out, status = Open3.capture2e('clang', '-x', 'c', '-o', probe, '-', stdin_data: 'int main(void) { return 0; }')
raise "Clang probe failed: #{out}" unless status.success?
artifacts << { 'path' => probe, 'goarch' => 'fresh-clang-host' }
results = []
signer = NativePackages::MacosSigning.new(Pathname.pwd)
signer.with_identity do
  keychain = signer.instance_variable_get(:@keychain)
  identities = signer.execute('security', 'find-identity', '-v', '-p', 'codesigning', keychain)
  fingerprint = identities[/\b[0-9A-F]{40}\b/]
  raise 'No imported identity fingerprint' unless fingerprint
  original_list = Shellwords.shellsplit(signer.execute('security', 'list-keychains', '-d', 'user'))
  puts JSON.generate(owned_keychain_in_original_search_list: original_list.include?(keychain.to_s))
  begin
    %w[original no_preserve fingerprint search_list search_list_no_preserve].each do |variant|
      if variant.start_with?('search_list')
        current_list = Shellwords.shellsplit(signer.execute('security', 'list-keychains', '-d', 'user'))
        signer.execute('security', 'list-keychains', '-d', 'user', '-s', *([keychain.to_s] + current_list).uniq)
      end
      artifacts.each do |artifact|
        Dir.mktmpdir('mqtt-codesign-diagnostic-') do |directory|
          binary = File.join(directory, 'mqtt-alive-daemon')
          FileUtils.cp(artifact.fetch('path'), binary)
          command = ['codesign', '--force', '--timestamp', '--options', 'runtime']
          command << '--preserve-metadata=identifier,entitlements,requirements' unless variant.end_with?('no_preserve')
          command += ['--keychain', keychain, '--sign', variant == 'fingerprint' ? fingerprint : ENV.fetch('APPLE_SIGNING_IDENTITY'), binary]
          result = { variant: variant, arch: artifact.fetch('goarch') }
          begin
            signer.execute(*command)
            signer.execute('codesign', '--verify', '--strict', binary)
            result[:success] = true
          rescue NativePackages::Error => error
            result.merge!(success: false, error: error.message)
          end
          results << result
          puts JSON.generate(result)
        end
      end
    end
  ensure
    current_list = Shellwords.shellsplit(signer.execute('security', 'list-keychains', '-d', 'user'))
    signer.execute('security', 'list-keychains', '-d', 'user', '-s', *current_list.reject { |path| path == keychain.to_s })
  end
end
File.write('dist/signing-diagnostic.json', JSON.pretty_generate(results) + "\n")
FileUtils.remove_entry(probe_directory)
