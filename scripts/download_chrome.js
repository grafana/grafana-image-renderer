const { BrowserPlatform, Browser, install, resolveBuildId } = require('@puppeteer/browsers');
const fs = require('fs')
const path = require('path');

const archArg = process.argv[2];
let [
    // Should be one of linux, mac, win32, win64 as per options in BrowserFetcher but we reuse the same arch string
    // as for grpc download (ie darwin-x64-unknown) so we need to transform it a bit
    platform,
    arch,
] = archArg.split('-');

if (platform === 'win32' && arch === 'x64') {
    platform = BrowserPlatform.WIN64;
}

if (platform === 'darwin') {
    if (arch === 'arm64') {
        platform = BrowserPlatform.MAC_ARM;
    } else {
        platform = BrowserPlatform.MAC;
    }
}

const outputPath = path.resolve(process.cwd(), 'dist', process.argv[3] || `plugin-${archArg}`);

const browserVersion = Browser.CHROME;

async function download() {
    const buildId = await resolveBuildId(browserVersion, platform, 'latest');
    console.log(`Installing ${browserVersion} into ${outputPath}`);
    return install({
        cacheDir: outputPath,
        browser: browserVersion,
        platform,
        buildId,
    });
}

download().then(browser => {
    console.log(`${browserVersion} downloaded into:`, outputPath);

    const chromeInfo = { buildId: browser.buildId };
    return fs.writeFileSync(path.resolve(outputPath, 'chrome-info.json'), JSON.stringify(chromeInfo));
});
