import * as grpc from 'grpc';
import * as protoLoader from '@grpc/proto-loader';
import { Logger } from './logger';
import { Browser } from './browser';

const SERVER_ADDRESS = '127.0.0.1:50059';
const RENDERER_PROTO_PATH = __dirname + '/../proto/renderer.proto';

export const renderPackageDef = protoLoader.loadSync(RENDERER_PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

export const rendererProtoDescriptor = grpc.loadPackageDefinition(renderPackageDef);

export class GrpcPlugin {
  constructor(private log: Logger, private browser: Browser) {}

  start() {
    var server = new grpc.Server();

    const models: any = rendererProtoDescriptor.models;
    server.addService(models.Renderer.service, {
      render: this.render.bind(this),
    });

    server.bind(SERVER_ADDRESS, grpc.ServerCredentials.createInsecure());
    server.start();

    console.log(`1|1|tcp|${SERVER_ADDRESS}|grpc`);

    if (this.browser.chromeBin) {
      this.log.info('Renderer plugin started', 'chromeBin', this.browser.chromeBin);
    } else {
      this.log.info('Renderer plugin started');
    }
  }

  check(call, callback) {
    callback(null, { status: 'SERVING' });
  }

  async render(call, callback) {
    let req = call.request;

    let options = {
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
      let result = await this.browser.render(options);
      callback(null, { error: '' });
    } catch (err) {
      this.log.error('Render request failed', 'url', options.url, 'error', err);
      callback(null, { error: err.toString() });
    }
  }
}
