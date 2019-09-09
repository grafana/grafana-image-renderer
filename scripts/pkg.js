const childProcess = require('child_process');

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

childProcess.execSync(`./node_modules/.bin/pkg -t node10-${platform}-${arch} . --out-path dist/plugin-${archArg}`, {stdio: 'inherit'});
