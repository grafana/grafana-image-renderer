const { BrowserPlatform, Browser, install, resolveBuildId } = require('@puppeteer/browsers');
const childProcess = require('child_process');
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

// Clean up download folder if exists
childProcess.execFileSync('rm', ['-rf', `${outputPath}/chrome`]);

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

download().then(() => {
    console.log(`${browserVersion} downloaded into:`, outputPath);

    // All the chrome files should be in the plugin 'chrome' folder
    // This is used to set the puppeteer.executablePath
    const ext = platform === BrowserPlatform.WIN64 ? '.exe' : '';
    const out = childProcess.execFileSync('find', [`${outputPath}/chrome`, '-type', 'f', '-name', `chrome${ext}`, '-exec', 'dirname', '{}', '\;']);
    const chromeBinDir = out.toString().trim()
    
    console.log(`Moving ${chromeBinDir} content into ${outputPath}/chrome`);
    childProcess.execFileSync('mv', [chromeBinDir, `${outputPath}/tmp`]);
    childProcess.execFileSync('rm', ['-r', `${outputPath}/chrome`]);
    childProcess.execFileSync('mv', [`${outputPath}/tmp`, `${outputPath}/chrome`]);
});
