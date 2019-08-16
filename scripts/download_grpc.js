const path = require('path');
const childProcess = require('child_process');
const https = require('https');
const fs = require('fs');

// /grpc-precompiled-binaries/node/grpc/v1.11.3/node-v64-darwin-x64-unknown
const grpcPackageJson = require('grpc/package.json');

// Taken from https://github.com/mapbox/node-pre-gyp/blob/674fda7b28b86bb3b11889db86c15c4d06722d02/lib/util/versioning.js
// so we can evaluate templates used for node-pre-gyp in grpc package
function eval_template(template, opts) {
  Object.keys(opts).forEach(function(key) {
    var pattern = '{' + key + '}';
    while (template.indexOf(pattern) > -1) {
      template = template.replace(pattern, opts[key]);
    }
  });
  return template;
}

// This is taken from grpc package that uses node-pre-gyp to download or build native grpc core.

// This matches abi version for node10 which is latest version pkg supports (as it is latest LTS)
// see https://nodejs.org/en/download/releases/
const node_abi = 'node-v64';
const name = grpcPackageJson.name;
const version = grpcPackageJson.version;

const archString = process.argv[2];
const pluginDir =  `plugin-${archString}`;
// See https://console.cloud.google.com/storage/browser/node-precompiled-binaries.grpc.io/grpc/?project=grpc-testing
// for existing prebuild binaries (though there are only ones for newer version).
const [
  // linux, darwin, win32
  platform,
  // ia32, x64, arm, arm64
  arch,
  // unknown, glibc, musl
  libc,
] = archString.split('-');

const host = grpcPackageJson.binary.host;
const remote_path = eval_template(grpcPackageJson.binary.remote_path, { name, version });
const package_name = eval_template(grpcPackageJson.binary.package_name, { node_abi, platform, arch, libc });
const url = host + path.join(remote_path, package_name);

console.log(`Getting ${url}`);
new Promise((resolve, reject) => {
  const file = fs.createWriteStream(`plugin-${archString}/grpc_node.tar.gz`);
  https
    .get(url, function(response) {
      if (response.statusCode !== 200) {
        const err = new Error(`response.statusCode = ${response.statusCode}`);
        err.response = response;
        reject(err);
      }
      response.pipe(file).on('finish', () => {
        resolve();
      });
    })
    .on('error', e => reject(e));
})
  .then(() => {
    console.log(`Grpc module downloaded`);
    const dirName = package_name.split('.')[0];
    childProcess.execSync(`tar -xzf ${pluginDir}/grpc_node.tar.gz --directory ${pluginDir}`);
    childProcess.execSync(`mv ${pluginDir}/${dirName}/grpc_node.node ${pluginDir}/`);
    childProcess.execSync(`rm -rf ${pluginDir}/${dirName}`);
    childProcess.execSync(`rm -rf ${pluginDir}/grpc_node.tar.gz`);
    process.exit(0);
  })
  .catch(err => {
    console.error(err);
    process.exit(1);
  });
