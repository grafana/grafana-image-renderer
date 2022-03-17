import path = require("path");
const fs = require('fs');

const outputFolder = process.argv[2]
const commit = process.argv[3];

if (!outputFolder) {
  throw new Error('expected output folder as the first arg')
}

if (!commit) {
  throw new Error(`usage: 'yarn run create-gcom-plugin-json <COMMIT_HASH>'`);
}

const outputPath = path.join(outputFolder, 'plugin.json')
const rootPluginJsonPath = path.resolve('./plugin.json');


enum PluginVersion {
  'darwin-amd64' = 'darwin-amd64',
  'linux-amd64' = 'linux-amd64',
  'windows-amd64' = 'windows-amd64',
}

type PluginJson = {
  url: string;
  commit: string;
  download: Record<PluginVersion, { url: string; md5: string }>;
};

const pluginVersionToFileName: Record<PluginVersion, string> = {
  [PluginVersion['darwin-amd64']]: 'plugin-darwin-x64-unknown.zip',
  [PluginVersion['linux-amd64']]: 'plugin-linux-x64-glibc.zip',
  [PluginVersion['windows-amd64']]: 'plugin-win32-x64-unknown.zip',
};

const baseUrl = `https://github.com/grafana/grafana-image-renderer`;
const fileUrl = (release, fileName) => `${baseUrl}/releases/download/${release}/${fileName}`;

const axios = require('axios');

const getFileNamesToChecksumMap = async (releaseVersion: string): Promise<Record<string, string>> => {
  const res = await axios.get(fileUrl(releaseVersion, 'md5sums.txt'));

  if (typeof res.data !== 'string') {
    throw new Error('expected checksum data to be string');
  }

  const text = res.data as string;

  return text
    .split('\n')
    .map((l) => l.replaceAll(/\s+/g, ' ').split(' '))
    .filter((arr) => arr.length === 2)
    .reduce((acc, [checksum, artifact]) => {
      const artifactPrefix = 'artifacts/';
      if (artifact.startsWith(artifactPrefix)) {
        return { ...acc, [artifact.substring(artifactPrefix.length)]: checksum };
      } else {
        throw new Error(`expected artifact name to start with "artifact/". actual: ${artifact}`);
      }
    }, {});
};

const verifyChecksums = (map: Record<string, string>) => {
  const expectedFileNames = Object.values(pluginVersionToFileName);
  const fileNamesInChecksumMap = Object.keys(map);
  for (const expectedFileName of expectedFileNames) {
    if (!fileNamesInChecksumMap.includes(expectedFileName)) {
      throw new Error(`expected to find ${expectedFileName} in the checksum map. actual: [${fileNamesInChecksumMap.join(', ')}]`);
    }
  }
};

const getReleaseVersion = (): string => {
  const rootPluginJson = JSON.parse(fs.readFileSync(rootPluginJsonPath));
  const version = rootPluginJson?.info?.version;

  if (!version || typeof version !== 'string' || !version.length) {
    throw new Error(`expected to find value for "info.version" in root plugin.json (${rootPluginJsonPath})`);
  }
  return `v${version}`;
};

const createGcomPluginJson = (map: Record<string, string>, releaseVersion: string): PluginJson => ({
  url: baseUrl,
  commit: commit,
  download: Object.values(PluginVersion)
    .map((ver) => {
      const fileName = pluginVersionToFileName[ver];
      const md5 = map[fileName];
      if (!md5 || !md5.length) {
        throw new Error(`expected non-empty md5 checksum for plugin version ${ver} with filename ${fileName}`);
      }

      return { [ver]: { md5, url: fileUrl(releaseVersion, fileName) } };
    })
    .reduce((acc, next) => ({ ...acc, ...next }), {}) as PluginJson['download'],
});

const run = async () => {
  const releaseVersion = getReleaseVersion();
  console.log(`Creating gcom plugin json with version ${releaseVersion} and commit ${commit}`);

  const artifactsToChecksumMap = await getFileNamesToChecksumMap(releaseVersion);
  verifyChecksums(artifactsToChecksumMap);

  console.log(`Fetched artifact checksums ${JSON.stringify(artifactsToChecksumMap, null, 2)}`);

  const pluginJson = createGcomPluginJson(artifactsToChecksumMap, releaseVersion);
  if (!fs.existsSync(outputFolder)) {
    fs.mkdirSync(outputFolder)
  }
  fs.writeFileSync(outputPath, JSON.stringify(pluginJson, null, 2));

  console.log(`Done! Path: ${path.resolve(outputPath)}`)
};

run();
