import request from 'supertest';
import * as jwt from 'jsonwebtoken';
import * as fs from 'fs';

import { HttpServer } from './http-server';
import { ConsoleLogger } from '../logger';
import { ServiceConfig } from '../config';
import { createSanitizer } from '../sanitizer/Sanitizer';
import path from 'path';

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

const goldenFilesFolder = './src/test/integrations/testdata';
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
    const url = `${grafanaEndpoint}/${dashboardUid}?panelId=${panelIds.graph}&render=1&from=1698385036999&to=1698406636999`;
    const filePath = path.join(__dirname, 'graph.png');
    const response = await request(server.app)
      .get(`/render?url=${url}&timeout=5&renderKey=${renderKey}&domain=localhost&width=500&height=300&deviceScaleFactor=1&filePath=${filePath}`)
      .set('X-Auth-Token', '-');
    // .buffer()
    // .parse((res, callback) => {
    //   res.setEncoding('binary');
    //   res.data = '';
    //   res.on('data', (chunk) => {
    //     res.data += chunk;
    //   });
    //   res.on('end', () => {
    //     callback(null, Buffer.from(res.data, 'binary'));
    //   });
    // });

    fs.writeFileSync('graph.png', response.body);

    expect(response.statusCode).toEqual(200);
    // expect(response.headers['content-disposition']).to.be.equal('attachment; filename=resume.csv');
    // expect(response.headers['content-type']).to.be.equal('text/csv; charset=utf-8');
    // expect(response.headers['content-length']).to.not.equal('0');
    // console.log(response.body);
    expect(response.body).toEqual(fs.readFileSync(goldenFilesFolder + '/graph.png'));
  });
});
