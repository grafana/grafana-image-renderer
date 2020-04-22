import * as grpc from 'grpc';
import * as protoLoader from '@grpc/proto-loader';
import { GrpcPlugin } from '../../node-plugin';
import { Logger } from '../../logger';
import { PluginConfig } from '../../config';
import { createBrowser, Browser } from '../../browser';
import { RenderOptions } from '../../browser/browser';

const renderPackageDef = protoLoader.loadSync(__dirname + '/../../../proto/renderer.proto', {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const rendererProtoDescriptor = grpc.loadPackageDefinition(renderPackageDef);

export class RenderGRPCPluginV1 implements GrpcPlugin {
  constructor(private config: PluginConfig, private log: Logger) {}

  async grpcServer(server: grpc.Server) {
    const serviceDef = rendererProtoDescriptor['models']['Renderer']['service'];
    const service = new RendererGRPCServer(this.config, this.log);
    server.addService(serviceDef, service);
    service.start();
  }
}

class RendererGRPCServer {
  private browser: Browser;

  constructor(config: PluginConfig, private log: Logger) {
    this.browser = createBrowser(config.rendering, log);
  }

  async start() {
    await this.browser.start();
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
      await this.browser.render(options);
      callback(null, { error: '' });
    } catch (err) {
      this.log.error('Render request failed', 'url', options.url, 'error', err.toString());
      callback(null, { error: err.toString() });
    }
  }
}
