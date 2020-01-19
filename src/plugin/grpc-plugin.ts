import * as grpc from 'grpc';
import * as protoLoader from '@grpc/proto-loader';
import { Logger } from '../logger';
import { Browser, RenderOptions } from '../browser/browser';
import { PluginConfig } from '../config';

const RENDERER_PROTO_PATH = __dirname + '/../../proto/renderer.proto';
const HEALTH_PROTO_PATH = __dirname + '/../../proto/health.proto';

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
  constructor(private config: PluginConfig, private log: Logger, private browser: Browser) {}

  async start() {
    const server = new grpc.Server();
    const grpcHealthV1: any = healthProtoDescriptor['grpc']['health']['v1'];
    server.addService(grpcHealthV1.Health.service, {
      check: this.check.bind(this),
    });
    const models: any = rendererProtoDescriptor.models;
    server.addService(models.Renderer.service, {
      render: this.render.bind(this),
    });

    const address = `${this.config.plugin.grpc.host}:${this.config.plugin.grpc.port}`;
    const boundPortNumber = server.bind(address, grpc.ServerCredentials.createInsecure());
    if (boundPortNumber === 0) {
      throw new Error(`failed to bind address=${address}, boundPortNumber=${boundPortNumber}`);
    }

    server.start();

    console.log(`1|1|tcp|${this.config.plugin.grpc.host}:${boundPortNumber}|grpc`);

    await this.browser.start();

    if (this.config.rendering.chromeBin) {
      this.log.info(
        'Renderer plugin started',
        'grpcHost',
        this.config.plugin.grpc.host,
        'grpcPort',
        boundPortNumber,
        'chromeBin',
        this.config.rendering.chromeBin,
        'ignoreHTTPSErrors',
        this.config.rendering.ignoresHttpsErrors
      );
    } else {
      this.log.info('Renderer plugin started', 'ignoreHttpsErrors', this.config.rendering.ignoresHttpsErrors);
    }
  }

  check(call, callback) {
    callback(null, { status: 'SERVING' });
  }

  async render(call, callback) {
    const req = call.request;
    const options: RenderOptions = {
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
