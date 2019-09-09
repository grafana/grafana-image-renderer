const path = require('path');
const child_process = require('child_process');
const Puppeteer = require('puppeteer');
const puppeteerPackageJson = require('puppeteer/package.json');

const archArg = process.argv[2];
const pluginDir = `dist/plugin-${archArg}`;
let [
  // Should be one of linux, mac, win32, win64 as per options in BrowserFetcher but we reuse the same arch string
  // as for grpc download (ie darwin-x64-unknown) so we need to transform it a bit
  platform,
  arch
] = archArg.split('-');

if (platform === 'win32' && arch === 'x64') {
  platform = 'win64'
}

if (platform === 'darwin') {
  platform = 'mac'
}

const browserFetcher = Puppeteer.createBrowserFetcher({ platform });
const revision = puppeteerPackageJson.puppeteer.chromium_revision;

browserFetcher
  .download(revision, null)
  .then(() => {
    console.log("Chromium downloaded");
    const parts = browserFetcher.revisionInfo(revision).executablePath.split(path.sep);

      // based on where puppeteer puts the binaries see BrowserFetcher.revisionInfo()
    while (!parts[parts.length - 1].startsWith('chrome-')) {
      parts.pop()
    }

    let execPath = parts.join(path.sep);

    child_process.execSync(`cp -RP ${execPath} ${pluginDir}`);

    console.log(`Chromium moved from ${execPath} to ${pluginDir}/`);
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
