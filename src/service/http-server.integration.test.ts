import * as fs from 'fs';
import * as jwt from 'jsonwebtoken';
import * as path from 'path';
import * as request from 'supertest';

import { HttpServer } from './http-server';
import { ConsoleLogger } from '../logger';
import { ServiceConfig } from './config';
import { createSanitizer } from '../sanitizer/Sanitizer';

const dashboardUid = 'd10881ec-0d35-4909-8de7-6ab563a9ab29';
const panelIds = {
  graph: 1,
  table: 2,
  error: 3,
  slow: 4,
};
const grafanaEndpoint = 'http://localhost:3000/d-solo';
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

const goldenFilesFolder = './src/testdata';
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
    verboseLogging: true,
    dumpio: true,
    timingMetrics: false,
    emulateNetworkConditions: false,
    // Uncoment to see what's happening in the browser during the tests
    // headed: true,
  },
};

const sanitizer = createSanitizer();
const server = new HttpServer(serviceConfig, new ConsoleLogger(serviceConfig.service.logging), sanitizer);

beforeAll(() => {
  process.env['PUPPETEER_DISABLE_HEADLESS_WARNING'] = 'true';
  return server.start();
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
    const url = `${grafanaEndpoint}/${dashboardUid}?panelId=${panelIds.graph}&render=1&from=1699333200000&to=1699344000000`;
    const response = await request(server.app)
      .get(`/render?url=${encodeURIComponent(url)}&timeout=5&renderKey=${renderKey}&domain=localhost&width=500&height=300&deviceScaleFactor=1`)
      .set('X-Auth-Token', '-');

    const goldenFilePath = path.join(goldenFilesFolder, 'graph.png');
    if (process.env['UPDATE_GOLDEN'] === 'true') {
      fs.writeFileSync(goldenFilePath, response.body);
    }

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');
    expect(response.body).toEqual(fs.readFileSync(goldenFilePath));
  });

  it('should respond with the table panel screenshot', async () => {
    const url = `${grafanaEndpoint}/${dashboardUid}?panelId=${panelIds.table}&render=1&from=1699333200000&to=1699344000000`;
    const response = await request(server.app)
      .get(`/render?url=${encodeURIComponent(url)}&timeout=5&renderKey=${renderKey}&domain=localhost&width=500&height=300&deviceScaleFactor=1`)
      .set('X-Auth-Token', '-');

    const goldenFilePath = path.join(goldenFilesFolder, 'table.png');
    if (process.env['UPDATE_GOLDEN'] === 'true') {
      fs.writeFileSync(goldenFilePath, response.body);
    }

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');
    expect(response.body).toEqual(fs.readFileSync(goldenFilePath));
  });

  it('should respond with a panel error screenshot', async () => {
    const url = `${grafanaEndpoint}/${dashboardUid}?panelId=${panelIds.error}&render=1&from=1699333200000&to=1699344000000`;
    const response = await request(server.app)
      .get(`/render?url=${encodeURIComponent(url)}&timeout=5&renderKey=${renderKey}&domain=localhost&width=500&height=300&deviceScaleFactor=1`)
      .set('X-Auth-Token', '-');

    const goldenFilePath = path.join(goldenFilesFolder, 'error.png');
    if (process.env['UPDATE_GOLDEN'] === 'true') {
      fs.writeFileSync(goldenFilePath, response.body);
    }

    expect(response.statusCode).toEqual(200);
    expect(response.headers['content-type']).toEqual('image/png');
    expect(response.body).toEqual(fs.readFileSync(goldenFilePath));
  });

  // TODO: this test currently doesn't pass because it takes a screenshot of the panel still loading and the position of the loading bar and the loading icon can vary
  // it('should timeout when a panel is too slow to load', async () => {
  //   const url = `${grafanaEndpoint}/${dashboardUid}?panelId=${panelIds.slow}&render=1&from=1699333200000&to=1699344000000`;
  //   const response = await request(server.app)
  //     .get(`/render?url=${encodeURIComponent(url)}&timeout=5&renderKey=${renderKey}&domain=localhost&width=500&height=300&deviceScaleFactor=1`)
  //     .set('X-Auth-Token', '-');

  //   const goldenFilePath = path.join(goldenFilesFolder, 'slow.png');
  //   if (process.env['UPDATE_GOLDEN'] === 'true') {
  //     fs.writeFileSync(goldenFilePath, response.body);
  //   }

  //   expect(response.statusCode).toEqual(200);
  //   expect(response.headers['content-type']).toEqual('image/png');
  //   expect(response.body).toEqual(fs.readFileSync(goldenFilePath));
  // }, 30000);
});
