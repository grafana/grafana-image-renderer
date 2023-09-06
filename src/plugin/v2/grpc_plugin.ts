import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import * as promClient from 'prom-client';
import { GrpcPlugin } from '../../node-plugin';
import { Logger } from '../../logger';
import { PluginConfig, SecurityConfig, isAuthTokenValid } from '../../config';
import { createBrowser, Browser } from '../../browser';
import { HTTPHeaders, ImageRenderOptions, RenderOptions } from '../../types';
import {
  RenderRequest,
  RenderResponse,
  RenderCSVRequest,
  RenderCSVResponse,
  CheckHealthRequest,
  CheckHealthResponse,
  CollectMetricsRequest,
  CollectMetricsResponse,
  HealthStatus,
  GRPCSanitizeRequest,
  GRPCSanitizeResponse,
} from './types';
import { createSanitizer, Sanitizer } from '../../sanitizer/Sanitizer';
import { SanitizeRequest } from '../../sanitizer/types';
import { Status } from '@grpc/grpc-js/build/src/constants';

const rendererV2PackageDef = protoLoader.loadSync(__dirname + '/../../../proto/rendererv2.proto', {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

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

const rendererV2ProtoDescriptor = grpc.loadPackageDefinition(rendererV2PackageDef);
const pluginV2ProtoDescriptor = grpc.loadPackageDefinition(pluginV2PackageDef);
const sanitizerProtoDescriptor = grpc.loadPackageDefinition(sanitizerPackageDef);

export class RenderGRPCPluginV2 implements GrpcPlugin {
  constructor(private config: PluginConfig, private log: Logger) {
    populateConfigFromEnv(this.config);
  }

  async grpcServer(server: grpc.Server) {
    const metrics = setupMetrics();
    const browser = createBrowser(this.config.rendering, this.log, metrics);
    const pluginService = new PluginGRPCServer(browser, this.log, createSanitizer(), this.config.plugin.security);

    const rendererServiceDef = rendererV2ProtoDescriptor['pluginextensionv2']['Renderer']['service'];
    server.addService(rendererServiceDef, pluginService as any);

    const pluginServiceDef = pluginV2ProtoDescriptor['pluginv2']['Diagnostics']['service'];
    server.addService(pluginServiceDef, pluginService as any);

    const sanitizerServiceDef = sanitizerProtoDescriptor['pluginextensionv2']['Sanitizer']['service'];
    server.addService(sanitizerServiceDef, pluginService as any);

    metrics.up.set(1);

    let browserVersion = 'unknown';
    let labelValue = 1;

    try {
      browserVersion = await browser.getBrowserVersion();
    } catch (err) {
      this.log.error('Failed to get browser version', 'err', err);
      labelValue = 0;
    }
    metrics.browserInfo.labels(browserVersion).set(labelValue);
    if (browserVersion !== 'unknown') {
      this.log.debug('Using browser version', 'browserVersion', browserVersion);
    }

    await pluginService.start(browserVersion);
  }
}

class PluginGRPCServer {
  private browserVersion: string | undefined;

  constructor(private browser: Browser, private log: Logger, private sanitizer: Sanitizer, private securityCfg: SecurityConfig) { }

  async start(browserVersion?: string) {
    this.browserVersion = browserVersion;
    await this.browser.start();
  }

  async render(call: grpc.ServerUnaryCall<RenderRequest, any>, callback: grpc.sendUnaryData<RenderResponse>) {
    const req = call.request;
    const headers: HTTPHeaders = {};

    if (!req) {
      return callback({ code: Status.INVALID_ARGUMENT, details: 'Request cannot be null' });
    }

    if (!isAuthTokenValid(this.securityCfg, req.authToken)) {
      return callback({ code: Status.UNAUTHENTICATED, details: 'Unauthorized request' });
    }

    if (req.url && !(req.url.startsWith('http://') || req.url.startsWith('https://'))) {
      return callback({ code: Status.INVALID_ARGUMENT, details: 'Forbidden query url protocol' });
    }

    if (req.headers) {
      for (const key in req.headers) {
        if (req.headers.hasOwnProperty(key)) {
          const h = req.headers[key];
          headers[key] = h.values.join(';');
        }
      }
    }

    const options: ImageRenderOptions = {
      url: req.url,
      width: req.width,
      height: req.height,
      filePath: req.filePath,
      timeout: req.timeout,
      renderKey: req.renderKey,
      domain: req.domain,
      timezone: req.timezone,
      deviceScaleFactor: req.deviceScaleFactor,
      headers: headers,
    };

    this.log.debug('Render request received', 'url', options.url);
    let errStr = '';
    try {
      await this.browser.render(options);
    } catch (err) {
      this.log.error('Render request failed', 'url', options.url, 'error', err.toString());
      errStr = err.toString();
    }
    callback(null, { error: errStr });
  }

  async renderCsv(call: grpc.ServerUnaryCall<RenderCSVRequest, any>, callback: grpc.sendUnaryData<RenderCSVResponse>) {
    const req = call.request;
    const headers: HTTPHeaders = {};

    if (!req) {
      return callback({ code: Status.INVALID_ARGUMENT, details: 'Request cannot be null' });
    }

    if (!isAuthTokenValid(this.securityCfg, req.authToken)) {
      return callback({ code: Status.UNAUTHENTICATED, details: 'Unauthorized request' });
    }

    if (req.url && !(req.url.startsWith('http://') || req.url.startsWith('https://'))) {
      return callback({ code: Status.INVALID_ARGUMENT, details: 'Forbidden query url protocol' });
    }

    if (req.headers) {
      for (const key in req.headers) {
        if (req.headers.hasOwnProperty(key)) {
          const h = req.headers[key];
          headers[key] = h.values.join(';');
        }
      }
    }

    const options: RenderOptions = {
      url: req.url,
      filePath: req.filePath,
      timeout: req.timeout,
      renderKey: req.renderKey,
      domain: req.domain,
      timezone: req.timezone,
      headers: headers,
    };

    this.log.debug('Render request received', 'url', options.url);
    let errStr = '';
    let fileName = '';
    try {
      const result = await this.browser.renderCSV(options);
      fileName = result.fileName || '';
    } catch (err) {
      this.log.error('Render request failed', 'url', options.url, 'error', err.toString());
      errStr = err.toString();
    }
    callback(null, { error: errStr, fileName });
  }

  async checkHealth(_: grpc.ServerUnaryCall<CheckHealthRequest, any>, callback: grpc.sendUnaryData<CheckHealthResponse>) {
    const jsonDetails = Buffer.from(
      JSON.stringify({
        browserVersion: this.browserVersion,
      })
    );

    callback(null, { status: HealthStatus.OK, message: 'Success', jsonDetails: jsonDetails });
  }

  async collectMetrics(_: grpc.ServerUnaryCall<CollectMetricsRequest, any>, callback: grpc.sendUnaryData<CollectMetricsResponse>) {
    const payload = Buffer.from(await promClient.register.metrics());
    callback(null, { metrics: { prometheus: payload } });
  }

  async sanitize(call: grpc.ServerUnaryCall<GRPCSanitizeRequest, any>, callback: grpc.sendUnaryData<GRPCSanitizeResponse>) {
    const grpcReq = call.request;

    if (!isAuthTokenValid(this.securityCfg, grpcReq.authToken)) {
      return callback({ code: Status.UNAUTHENTICATED, details: 'Unauthorized request' });
    }

    const req: SanitizeRequest = {
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

  if (env['GF_PLUGIN_RENDERING_ARGS']) {
    const args = env['GF_PLUGIN_RENDERING_ARGS'] as string;
    if (args.length > 0) {
      const argsList = args.split(',');
      if (argsList.length > 0) {
        config.rendering.args = argsList;
      }
    }
  }

  if (env['GF_PLUGIN_RENDERING_IGNORE_HTTPS_ERRORS']) {
    config.rendering.ignoresHttpsErrors = env['GF_PLUGIN_RENDERING_IGNORE_HTTPS_ERRORS'] === 'true';
  }

  if (env['GF_PLUGIN_RENDERING_TIMEZONE']) {
    config.rendering.timezone = env['GF_PLUGIN_RENDERING_TIMEZONE'];
  } else {
    config.rendering.timezone = env['TZ'];
  }

  if (env['GF_PLUGIN_RENDERING_LANGUAGE']) {
    config.rendering.acceptLanguage = env['GF_PLUGIN_RENDERING_LANGUAGE'];
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_WIDTH']) {
    config.rendering.width = parseInt(env['GF_PLUGIN_RENDERING_VIEWPORT_WIDTH'] as string, 10);
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_HEIGHT']) {
    config.rendering.height = parseInt(env['GF_PLUGIN_RENDERING_VIEWPORT_HEIGHT'] as string, 10);
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR']) {
    config.rendering.deviceScaleFactor = parseFloat(env['GF_PLUGIN_RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR'] as string);
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_MAX_WIDTH']) {
    config.rendering.maxWidth = parseInt(env['GF_PLUGIN_RENDERING_VIEWPORT_MAX_WIDTH'] as string, 10);
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_MAX_HEIGHT']) {
    config.rendering.maxHeight = parseInt(env['GF_PLUGIN_RENDERING_VIEWPORT_MAX_HEIGHT'] as string, 10);
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR']) {
    config.rendering.maxDeviceScaleFactor = parseFloat(env['GF_PLUGIN_RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR'] as string);
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL']) {
    config.rendering.pageZoomLevel = parseFloat(env['GF_PLUGIN_RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL'] as string);
  }

  if (env['GF_PLUGIN_RENDERING_MODE']) {
    config.rendering.mode = env['GF_PLUGIN_RENDERING_MODE'] as string;
  }

  if (env['GF_PLUGIN_RENDERING_CLUSTERING_MODE']) {
    config.rendering.clustering.mode = env['GF_PLUGIN_RENDERING_CLUSTERING_MODE'] as string;
  }

  if (env['GF_PLUGIN_RENDERING_CLUSTERING_MAX_CONCURRENCY']) {
    config.rendering.clustering.maxConcurrency = parseInt(env['GF_PLUGIN_RENDERING_CLUSTERING_MAX_CONCURRENCY'] as string, 10);
  }

  if (env['GF_PLUGIN_RENDERING_CLUSTERING_TIMEOUT']) {
    config.rendering.clustering.timeout = parseInt(env['GF_PLUGIN_RENDERING_CLUSTERING_TIMEOUT'] as string, 10);
  }

  if (env['GF_PLUGIN_RENDERING_VERBOSE_LOGGING']) {
    config.rendering.verboseLogging = env['GF_PLUGIN_RENDERING_VERBOSE_LOGGING'] === 'true';
  }

  if (env['GF_PLUGIN_RENDERING_DUMPIO']) {
    config.rendering.dumpio = env['GF_PLUGIN_RENDERING_DUMPIO'] === 'true';
  }

  if (env['GF_PLUGIN_AUTH_TOKEN']) {
    const authToken = env['GF_PLUGIN_AUTH_TOKEN'] as string;
    config.plugin.security.authToken = authToken.includes(' ') ? authToken.split(' ') : authToken;
  }
};

interface PluginMetrics {
  up: promClient.Gauge;
  browserInfo: promClient.Gauge;
  durationHistogram: promClient.Histogram;
}

const setupMetrics = (): PluginMetrics => {
  promClient.collectDefaultMetrics();

  return {
    up: new promClient.Gauge({
      name: 'up',
      help: '1 = up, 0 = not up',
    }),
    browserInfo: new promClient.Gauge({
      name: 'grafana_image_renderer_browser_info',
      help: "A metric with a constant '1 value labeled by version of the browser in use",
      labelNames: ['version'],
    }),
    durationHistogram: new promClient.Histogram({
      name: 'grafana_image_renderer_step_duration_seconds',
      help: 'duration histogram of browser steps for rendering an image labeled with: step',
      labelNames: ['step'],
      buckets: [0.1, 0.3, 0.5, 1, 3, 5, 10, 20, 30],
    }),
  };
};
