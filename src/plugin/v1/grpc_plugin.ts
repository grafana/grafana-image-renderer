import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import { GrpcPlugin } from '../../node-plugin';
import { Logger } from '../../logger';
import { PluginConfig } from '../../config';
import { createBrowser, Browser } from '../../browser';
import { RenderOptions } from '../../browser/browser';
import CancellationToken from 'cancellationtoken';

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
    const browser = createBrowser(this.config.rendering, this.log);
    server.addService(serviceDef, {
      render: (call: grpc.ServerUnaryCall<any, any>, callback) => {
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

        const { cancel, token } = CancellationToken.create();

        this.log.debug('Render request received', 'url', options.url);
        browser.render(token, options).then(
          () => {
            callback(null, { error: '' });
          },
          err => {
            this.log.error('Render request failed', 'url', options.url, 'error', err.toString());
            callback(null, { error: err.toString() });
          }
        );
      },
    });
    await browser.start();
  }
}
