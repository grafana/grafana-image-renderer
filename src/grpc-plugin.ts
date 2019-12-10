import * as grpc from 'grpc';
import * as protoLoader from '@grpc/proto-loader';
import { Logger } from './logger';
import { Browser } from './browser';

const SERVER_ADDRESS = '127.0.0.1:50059';
const RENDERER_PROTO_PATH = __dirname + '/../proto/renderer.proto';
const HEALTH_PROTO_PATH = __dirname + '/../proto/health.proto';

export const renderPackageDef = protoLoader.loadSync(RENDERER_PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

export const healthPackageDef = protoLoader.loadSync(HEALTH_PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

export const rendererProtoDescriptor = grpc.loadPackageDefinition(renderPackageDef);
export const healthProtoDescriptor = grpc.loadPackageDefinition(healthPackageDef);

export class GrpcPlugin {
  constructor(private log: Logger, private browser: Browser) {}

  start() {
    const server = new grpc.Server();

    const grpcHealthV1: any = healthProtoDescriptor['grpc']['health']['v1'];
    server.addService(grpcHealthV1.Health.service, {
      check: this.check.bind(this),
    });
    const models: any = rendererProtoDescriptor.models;
    server.addService(models.Renderer.service, {
      render: this.render.bind(this),
    });

    server.bind(SERVER_ADDRESS, grpc.ServerCredentials.createInsecure());
    server.start();

    console.log(`1|1|tcp|${SERVER_ADDRESS}|grpc`);

    if (this.browser.chromeBin) {
      this.log.info('Renderer plugin started', 'chromeBin', this.browser.chromeBin, 'ignoreHTTPSErrors', this.browser.ignoreHTTPSErrors);
    } else {
      this.log.info('Renderer plugin started', 'ignoreHttpsErrors', this.browser.ignoreHTTPSErrors);
    }
  }

  check(call, callback) {
    callback(null, { status: 'SERVING' });
  }

  async render(call, callback) {
    const req = call.request;
    const options = {
      url: req.url,
      width: req.width,
      height: req.height,
      filePath: req.filePath,
      timeout: req.timeout,
      renderKey: req.renderKey,
      domain: req.domain,
      timezone: req.timezone,
      encoding: req.encoding,
    };

    try {
      this.log.debug('Render request received', 'url', options.url);
      const result = await this.browser.render(options);
      callback(null, { error: '' });
    } catch (err) {
      this.log.error('Render request failed', 'url', options.url, 'error', err);
      callback(null, { error: err.toString() });
    }
  }
}
