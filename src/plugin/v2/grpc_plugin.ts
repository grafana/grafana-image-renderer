import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as promClient from 'prom-client';
import { GrpcPlugin } from '../../node-plugin';
import { Logger } from '../../logger';
import { PluginConfig } from '../../config';
import { SanitizeRequestV2 } from '../../types';
import {
  CheckHealthRequest,
  CheckHealthResponse,
  CollectMetricsRequest,
  CollectMetricsResponse,
  HealthStatus,
  GRPCSanitizeRequest,
  GRPCSanitizeResponse,
} from './types';
import { createSanitizer, Sanitizer } from '../../sanitizer/Sanitizer';

const pluginV2PackageDef = protoLoader.loadSync(__dirname + '/../../../proto/pluginv2.proto', {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const sanitizerPackageDef = protoLoader.loadSync(__dirname + '/../../../proto/sanitizer.proto', {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const pluginV2ProtoDescriptor = grpc.loadPackageDefinition(pluginV2PackageDef);
const sanitizerProtoDescriptor = grpc.loadPackageDefinition(sanitizerPackageDef);

export class RenderGRPCPluginV2 implements GrpcPlugin {
  constructor(private config: PluginConfig, private log: Logger) {
    populateConfigFromEnv(this.config);
  }

  async grpcServer(server: grpc.Server) {
    const metrics = setupMetrics();
    const pluginService = new PluginGRPCServer(this.log, createSanitizer());

    const pluginServiceDef = pluginV2ProtoDescriptor['pluginv2']['Diagnostics']['service'];
    server.addService(pluginServiceDef, pluginService as any);

    const sanitizerServiceDef = sanitizerProtoDescriptor['pluginextensionv2']['Sanitizer']['service'];
    server.addService(sanitizerServiceDef, pluginService as any);

    metrics.up.set(1);
  }
}

class PluginGRPCServer {
  private browserVersion: string | undefined;

  constructor(private log: Logger, private sanitizer: Sanitizer) {}

  async checkHealth(_: grpc.ServerUnaryCall<CheckHealthRequest, any>, callback: grpc.sendUnaryData<CheckHealthResponse>) {
    const jsonDetails = Buffer.from(
      JSON.stringify({
        browserVersion: this.browserVersion,
      })
    );

    callback(null, { status: HealthStatus.OK, message: 'Success', jsonDetails: jsonDetails });
  }

  async collectMetrics(_: grpc.ServerUnaryCall<CollectMetricsRequest, any>, callback: grpc.sendUnaryData<CollectMetricsResponse>) {
    const payload = Buffer.from(promClient.register.metrics());
    callback(null, { metrics: { prometheus: payload } });
  }

  async sanitize(call: grpc.ServerUnaryCall<GRPCSanitizeRequest, any>, callback: grpc.sendUnaryData<GRPCSanitizeResponse>) {
    const grpcReq = call.request;

    const req: SanitizeRequestV2 = {
      content: grpcReq.content,
      config: JSON.parse(grpcReq.config.toString()),
      configType: grpcReq.configType,
    };

    this.log.debug('Sanitize request received', 'contentLength', req.content.length, 'name', grpcReq.filename);

    try {
      const sanitizeResponse = this.sanitizer.sanitize(req);
      callback(null, { error: '', sanitized: sanitizeResponse.sanitized });
    } catch (e) {
      this.log.error('Sanitization failed', 'contentLength', req.content.length, 'name', grpcReq.filename, 'error', e.stack);
      callback(null, { error: e.stack, sanitized: Buffer.from('', 'binary') });
    }
  }
}

const populateConfigFromEnv = (config: PluginConfig) => {
  const env = Object.assign({}, process.env);

  if (env['GF_PLUGIN_SANITIZER_VERBOSE_LOGGING']) {
    config.rendering.verboseLogging = env['GF_PLUGIN_RENDERING_VERBOSE_LOGGING'] === 'true';
  }
};

interface PluginMetrics {
  up: promClient.Gauge;
  durationHistogram: promClient.Histogram;
}

const setupMetrics = (): PluginMetrics => {
  promClient.collectDefaultMetrics();

  return {
    up: new promClient.Gauge({
      name: 'up',
      help: '1 = up, 0 = not up',
    }),

    durationHistogram: new promClient.Histogram({
      name: 'grafana_image_renderer_step_duration_seconds',
      help: 'duration histogram of browser steps for rendering an image labeled with: step',
      labelNames: ['step'],
      buckets: [0.1, 0.3, 0.5, 1, 3, 5, 10, 20, 30],
    }),
  };
};
