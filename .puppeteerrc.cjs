const {join} = require('path');
const path = require('path')

const cacheDirectory  = '/Users/alexk/dev/go/grafana-image-renderer/dist/chrome/mac_arm-114.0.5735.90/chrome-mac-arm64'// path.join(__dirname, 'dist', 'chrome');

console.log('cacheDir', cacheDirectory );
/**
 * @type {import("puppeteer").Configuration}
 */
module.exports = {
    // Changes the cache location for Puppeteer.
    cacheDirectory
};