import * as fs from 'fs';
import * as jwt from 'jsonwebtoken';
import * as path from 'path';
import * as request from 'supertest';
import * as pixelmatch from 'pixelmatch';
import * as fastPng from 'fast-png';

import { HttpServer } from './http-server';
import { ConsoleLogger } from '../logger';
import { ServiceConfig } from './config';
import { createSanitizer } from '../sanitizer/Sanitizer';

const testDashboardUid = 'd10881ec-0d35-4909-8de7-6ab563a9ab29';
const allPanelsDashboardUid = 'edlopzu6hn4lcd';
const panelIds = {
  graph: 1,
  table: 2,
  error: 3,
  slow: 4,
};
const renderKey = jwt.sign(
  {
    renderUser: {
      org_id: 1,
      user_id: 1,
      org_role: 'Admin',
    },
  },
  '-',
  { algorithm: 'HS512' }
);

const goldenFilesFolder = './tests/testdata';
const serviceConfig: ServiceConfig = {
  service: {
    host: undefined,
    port: 8081,
    metrics: {
      enabled: false,
      collectDefaultMetrics: true,
      requestDurationBuckets: [0.5, 1, 3, 5, 7, 10, 20, 30, 60],
    },
    logging: {
      level: 'debug',
      console: {
        json: true,
        colorize: false,
      },
    },
    security: {
      authToken: '-',
    },
  },
  rendering: {
    args: ['--no-sandbox', '--disable-gpu'],
    ignoresHttpsErrors: false,
    timezone: 'Europe/Paris',
    width: 500,
    height: 300,
    deviceScaleFactor: 1,
    maxWidth: 1000,
    maxHeight: 500,
    maxDeviceScaleFactor: 2,
    pageZoomLevel: 1,
    mode: 'default',
    clustering: {
      monitor: false,
      mode: 'browser',
      maxConcurrency: 5,
      timeout: 30,
    },
    timingMetrics: false,
    emulateNetworkConditions: false,
    // Set to true to get more logs
    verboseLogging: false, // true,
    dumpio: false, // true,
    // Uncoment to see what's happening in the browser during the tests
    // headed: true,
  },
};

const imageWidth = 500;
const imageHeight = 300;
const imageDiffThreshold = 0.01;
const matchingThreshold = 0.3;

const sanitizer = createSanitizer();
const server = new HttpServer(serviceConfig, new ConsoleLogger(serviceConfig.service.logging), sanitizer);

let domain = 'localhost';
function getGrafanaEndpoint(domain: string) {
  return `http://${domain}:3000`;
}

let envSettings = {
  saveDiff: false,
  updateGolden: false,
}

beforeAll(() => {
  process.env['PUPPETEER_DISABLE_HEADLESS_WARNING'] = 'true';

  if (process.env['CI'] === 'true') {
    domain = 'grafana';
  }
  
  envSettings.saveDiff = process.env['SAVE_DIFF'] === 'true'
  envSettings.updateGolden = process.env['UPDATE_GOLDEN'] === 'true'

  return server.start();
});

afterAll(() => {
  return server.close();
});

describe('Test /render/version', () => {
  it('should respond with unauthorized', () => {
    return request(server.app).get('/render/version').expect(401);
  });

  it('should respond with the current plugin version', () => {
    const pluginInfo = require('../../plugin.json');

    return request(server.app).get('/render/version').set('X-Auth-Token', '-').expect(200, { version: pluginInfo.info.version });
  });
});

describe('Test /render', () => {
  it('should respond with unauthorized', () => {
    return request(server.app).get('/render').expect(401);
  });

  it('should respond with the graph panel screenshot', async () => {
    const url = `${getGrafanaEndpoint(domain)}/d-solo/${testDashboardUid}?panelId=${panelIds.graph}&render=1&from=1699333200000&to=1699344000000`;
    const response = await request(server.app)
      .get(
        `/render?url=${encodeURIComponent(
          url
        )}&timeout=5&renderKey=${renderKey}&domain=${domain}&width=${imageWidth}&height=${imageHeight}&deviceScaleFactor=1`
      )
      .set('X-Auth-Token', '-');

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');

    const ok = compareImage('graph', response.body);
    expect(ok).toBeTruthy();
  });

  it('should respond with the table panel screenshot', async () => {
    const url = `${getGrafanaEndpoint(domain)}/d-solo/${testDashboardUid}?panelId=${panelIds.table}&render=1&from=1699333200000&to=1699344000000`;
    const response = await request(server.app)
      .get(
        `/render?url=${encodeURIComponent(
          url
        )}&timeout=5&renderKey=${renderKey}&domain=${domain}&width=${imageWidth}&height=${imageHeight}&deviceScaleFactor=1`
      )
      .set('X-Auth-Token', '-');

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');

    const ok = compareImage('table', response.body);
    expect(ok).toBeTruthy();
  });

  it('should respond with a panel error screenshot', async () => {
    const url = `${getGrafanaEndpoint(domain)}/d-solo/${testDashboardUid}?panelId=${panelIds.error}&render=1&from=1699333200000&to=1699344000000`;
    const response = await request(server.app)
      .get(
        `/render?url=${encodeURIComponent(
          url
        )}&timeout=5&renderKey=${renderKey}&domain=${domain}&width=${imageWidth}&height=${imageHeight}&deviceScaleFactor=1`
      )
      .set('X-Auth-Token', '-');

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');

    const ok = compareImage('error', response.body);
    expect(ok).toBeTruthy();
  });

  
  it('should take a full dashboard screenshot', async () => {
    const url = `${getGrafanaEndpoint(domain)}/d/${allPanelsDashboardUid}?render=1&from=1699333200000&to=1699344000000&kiosk=true`;
    const response = await request(server.app)
      .get(
        `/render?url=${encodeURIComponent(
          url
        )}&timeout=5&renderKey=${renderKey}&domain=${domain}&width=${imageWidth}&height=-1&deviceScaleFactor=1`
      )
      .set('X-Auth-Token', '-');

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');

    const ok = compareImage('full-page-screenshot', response.body);
    expect(ok).toBeTruthy();
  });
});

// compareImage returns if the number of different pixels between the image stored in the test file 
//   and the one from the response body is lower than a threshold (0.1 * image size)
// It updates the stored file and returns 0 if tests are run with UPDATE_GOLDEN=true.
// It writes the diff file to /testdata if tests are run with SAVE_DIFF=true.
function compareImage(testName: string, responseBody: any): boolean {
  const goldenFilePath = path.join(goldenFilesFolder, `${testName}.png`);
  if (envSettings.updateGolden) {
    fs.writeFileSync(goldenFilePath, responseBody);
    return true;
  }

  const expectedImage = fastPng.decode(fs.readFileSync(goldenFilePath));
  const { width, height } = expectedImage

  let diff: { width: number; height: number; data: Uint8ClampedArray } | null = null;
  if (envSettings.saveDiff) {
    diff = {
      width: width,
      height: height,
      data: new Uint8ClampedArray(width * height * 4),
    };
  }

  const responseImage = fastPng.decode(responseBody);

  const pixelDiff = pixelmatch(
    responseImage.data as Uint8ClampedArray,
    expectedImage.data as Uint8ClampedArray,
    diff ? diff.data : null,
    width,
    height,
    {
      threshold: matchingThreshold,
    }
  );

  const imagePixelDiffThreshold = imageDiffThreshold * width * height;
  if (diff && pixelDiff >= imagePixelDiffThreshold) {
    fs.writeFileSync(path.join(goldenFilesFolder, `diff_${testName}.png`), fastPng.encode(diff as fastPng.ImageData));
  }

  console.log('threshold: ', imagePixelDiffThreshold)
  console.log('pixelDiff: ', pixelDiff)

  return pixelDiff <= imagePixelDiffThreshold;
}
