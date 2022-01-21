const fs = require('fs');
const sharp = require('sharp');
const path = process.argv[2];
const imageExists = fs.existsSync(path);

console.log(JSON.stringify({ imageExists, path }));

if (imageExists) {
  sharp(path).resize(30, 20).toFile('./resized.webp');
}
