const path = require('path');
const child_process = require('child_process');
const Puppeteer = require('puppeteer');
const puppeteerPackageJson = require('puppeteer/package.json');


// linux, mac, win32, win64 as per options in BrowserFetcher
const platform = process.argv[2];
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

    child_process.execSync(`cp -RP ${execPath} plugin`);

    console.log(`Chromium moved from ${execPath} to plugin/`);
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
