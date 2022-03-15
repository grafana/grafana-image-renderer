const path = require('path');
const child_process = require('child_process');
const Puppeteer = require('puppeteer');
const { PUPPETEER_REVISIONS } = require('puppeteer/lib/cjs/puppeteer/revisions')
const fs = require('fs')

const archArg = process.argv[2];
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

const outputPath = "dist/" + (process.argv[3] || `plugin-${archArg}`);

const browserFetcher = Puppeteer.createBrowserFetcher({ platform });
const revision = PUPPETEER_REVISIONS.chromium;

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

    if (platform === 'mac' && arch === 'arm64') {
        // follow symlinks, dereference symlinks and copy them as files
        child_process.execSync(`cp -LR ${execPath} ${outputPath}`);

        const dsStore = '.DS_Store'
        const dsStorePaths = [`${outputPath}/${dsStore}`, `${outputPath}/${parts[parts.length - 1]}/.DS_Store`]
        for (const path of dsStorePaths) {
            if (!fs.existsSync(path)) {
                child_process.execSync(`touch ${path}`)
            }
        }
    } else {
        // follow symlinks, copy them as symlinks
        child_process.execSync(`cp -RP ${execPath} ${outputPath}`);
    }

    console.log(`Chromium moved from ${execPath} to ${outputPath}/`);
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
