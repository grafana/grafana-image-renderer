const childProcess = require('child_process');

const archArg = process.argv[2];

let [
  // linux, darwin, win32
  platform,
  // ia32, x64, arm, arm64
  arch,
] = archArg.split('-');

const platformTransform = {
  win32: 'windows',
  alpine: 'linux',
};

const archTransform = {
  x64: 'amd64',
  ia32: '386'
};

let ext = platform === 'win32' ? '.exe' : '';
const outputPath = "dist/" + (process.argv[3] || `plugin-${archArg}`);

const execFileName = `plugin_start_${platformTransform[platform] || platform}_${archTransform[arch] || arch}${ext}`;
childProcess.execSync(`mv ${outputPath}/renderer${ext} ${outputPath}/${execFileName}`, {stdio: 'inherit'});

