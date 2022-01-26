const child_process = require('child_process');
const fs = require('fs')

const archArg = process.argv[2];

// https://sharp.pixelplumbing.com/install#cross-platform
const [
    platform, // linux, darwin, win32
    arch // x64, ia32, arm, arm64
] = archArg.split('-');

const packageJson = JSON.parse(fs.readFileSync('./package.json'))

const packageName = 'sharp';
const packageVersion = Object.entries(packageJson.dependencies).find(([depName]) => depName === packageName)[1]

console.log(`Found ${packageName} with version ${packageVersion}`)

const targetNodeModules = './node_modules'
const currentSharp = `${targetNodeModules}/sharp`
const sharpDownloader = './scripts/sharp-downloader'

if(!fs.existsSync(sharpDownloader)) {
    fs.mkdirSync(sharpDownloader)
}

fs.writeFileSync(`${sharpDownloader}/package.json`, JSON.stringify({
        "name": "sharp-downloader",
        "dependencies": {
            [packageName]: packageVersion
        }
    }
))

const downloadedSharp = `${sharpDownloader}/node_modules/sharp`

try {
    child_process.execSync(`cd ${sharpDownloader} && npm install --platform=${platform} --arch=${arch}`, {stdio: 'inherit'})
    child_process.execSync(`rm -rf ${currentSharp} && cp -RP ${downloadedSharp} ${targetNodeModules}`);
} finally {
    fs.rmSync(sharpDownloader, {recursive: true, force: true})
}
