const childProcess = require('child_process');
const fs = require('fs')

const archArg = process.argv[2];
let [
  // linux, darwin, win32
  platform,
  // ia32, x64, arm, arm64
  arch,
] = archArg.split('-');

const platformTransform = {
  darwin: 'macos',
  win32: 'win',
};

const archTransform = {
  ia32: 'x84',
  arm: 'armv6',
  // I only assume this is correct
  arm64: 'armv6',
};

platform = platformTransform[platform] || platform;
arch = archTransform[arch] || arch;

if(platform === 'macos' && (arch.includes('arm'))) {
  arch = 'arm64'
}

const outputPath = "dist/" + (process.argv[3] || `plugin-${archArg}`);
const outputNodeModules = `${outputPath}/node_modules`

childProcess.execSync(`"./node_modules/.bin/pkg" -t node16-${platform}-${arch} . --out-path ${outputPath} --no-native-build`, {stdio: 'inherit'});

childProcess.execSync(`rm -rf ${outputNodeModules}`)
