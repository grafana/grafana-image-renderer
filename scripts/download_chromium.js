const path = require('path');
const child_process = require('child_process');
const Puppeteer = require('puppeteer');
const {BrowserPlatform, Browser, install, resolveBuildId} = require('@puppeteer/browsers')

const archArg = process.argv[2];
let [
    // Should be one of linux, mac, win32, win64 as per options in BrowserFetcher but we reuse the same arch string
    // as for grpc download (ie darwin-x64-unknown) so we need to transform it a bit
    platform,
    arch
] = archArg.split('-');

if (platform === 'win32' && arch === 'x64') {
    platform = BrowserPlatform.WIN64
}


if (platform === 'darwin') {
    platform = BrowserPlatform.MAC
}

//const outputPath = "dist/" + (process.argv[3] || `plugin-${archArg}`);
const outputPath = path.resolve(process.cwd(), "dist", process.argv[3] || `plugin-${archArg}`);

// const browserFetcher = Puppeteer.createBrowserFetcher({ platform });
//const revision = PUPPETEER_REVISIONS.chromium;


async function download() {
    const browserVersion = Browser.CHROME;
    const buildId = await resolveBuildId(browserVersion, platform, 'latest')
    console.log(`Installing ${browserVersion} into ${outputPath}`);
    return install({
        cacheDir: outputPath,
        browser: browserVersion,
        platform,
        buildId,
    })
}

download().then(() => {
    console.log('Chrome downloaded into:', outputPath);
})





