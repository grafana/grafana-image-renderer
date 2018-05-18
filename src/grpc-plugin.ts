import * as grpc from 'grpc';
import { Logger } from './logger';
import { Browser } from './browser';

const SERVER_ADDRESS = '127.0.0.1:50059';
const RENDERER_PROTO_PATH = __dirname + '/../proto/renderer.proto';
const GRPC_HEALTH_PROTO_PATH = __dirname + '/../proto/health.proto';

export const RENDERER_PROTO = grpc.load(RENDERER_PROTO_PATH).models;
export const GRPC_HEALTH_PROTO = grpc.load(GRPC_HEALTH_PROTO_PATH).grpc.health.v1;


export class GrpcPlugin {

  constructor(private log: Logger, private browser: Browser) {
  }

  start() {
    var server = new grpc.Server();

    server.addService(GRPC_HEALTH_PROTO.Health.service, {
      check: this.check.bind(this),
    });
    server.addService(RENDERER_PROTO.Renderer.service, {
      render: this.render.bind(this),
    });

    server.bind(SERVER_ADDRESS, grpc.ServerCredentials.createInsecure());
    server.start();

    console.log(`1|1|tcp|${SERVER_ADDRESS}|grpc`);
    this.log.info('Renderer plugin started');
  }

  check(call, callback) {
    callback(null, {status: 'SERVING'});
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
      let result = await this.browser.render(options);
      callback(null, {error: ''});
    } catch (err) {
      this.log.info("Error", err);
      callback(null, {error: err.toString()});
    }
  }
}

