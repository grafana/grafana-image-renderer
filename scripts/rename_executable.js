const childProcess = require('child_process');

const archArg = process.argv[2];

let [
  // linux, darwin, win32
  platform,
  // ia32, x64, arm, arm64
  arch,
] = archArg.split('-');

const platformTransform = {
  win32: 'windows'
};

const archTransform = {
  x64: 'amd64',
  ia32: '386'
};

let ext = platform === 'win32' ? '.exe' : '';

const execFileName = `plugin_start_${platformTransform[platform] || platform}_${archTransform[arch] || arch}${ext}`;
childProcess.execSync(`mv dist/plugin-${archArg}/renderer${ext} dist/plugin-${archArg}/${execFileName}`);

