import * as grpc from 'grpc';
import * as protoLoader from '@grpc/proto-loader';
import * as promClient from 'prom-client';
import { GrpcPlugin } from '../../node-plugin';
import { Logger } from '../../logger';
import { PluginConfig } from '../../config';
import { createBrowser, Browser } from '../../browser';
import { RenderOptions, HTTPHeaders } from '../../browser/browser';
import {
  RenderRequest,
  RenderResponse,
  CheckHealthRequest,
  CheckHealthResponse,
  CollectMetricsRequest,
  CollectMetricsResponse,
  HealthStatus,
} from './types';

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

const rendererV2ProtoDescriptor = grpc.loadPackageDefinition(rendererV2PackageDef);
const pluginV2ProtoDescriptor = grpc.loadPackageDefinition(pluginV2PackageDef);

export class RenderGRPCPluginV2 implements GrpcPlugin {
  constructor(private config: PluginConfig, private log: Logger) {
    populateConfigFromEnv(this.config);
  }

  async grpcServer(server: grpc.Server) {
    const metrics = setupMetrics();
    const browser = createBrowser(this.config.rendering, this.log);
    const pluginService = new PluginGRPCServer(browser, this.log, metrics);

    const rendererServiceDef = rendererV2ProtoDescriptor['pluginextensionv2']['Renderer']['service'];
    server.addService(rendererServiceDef, pluginService);

    const pluginServiceDef = pluginV2ProtoDescriptor['pluginv2']['Diagnostics']['service'];
    server.addService(pluginServiceDef, pluginService);

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

  constructor(private browser: Browser, private log: Logger, private metrics: Metrics) {}

  async start(browserVersion?: string) {
    this.browserVersion = browserVersion;
    await this.browser.start();
  }

  async render(call: grpc.ServerUnaryCall<RenderRequest>, callback: grpc.sendUnaryData<RenderResponse>) {
    const req = call.request;
    const headers: HTTPHeaders = {};

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

  async checkHealth(_: grpc.ServerUnaryCall<CheckHealthRequest>, callback: grpc.sendUnaryData<CheckHealthResponse>) {
    const jsonDetails = Buffer.from(
      JSON.stringify({
        browserVersion: this.browserVersion,
      })
    );

    callback(null, { status: HealthStatus.OK, message: 'Success', jsonDetails: jsonDetails });
  }

  async collectMetrics(_: grpc.ServerUnaryCall<CollectMetricsRequest>, callback: grpc.sendUnaryData<CollectMetricsResponse>) {
    const payload = Buffer.from(promClient.register.metrics());
    callback(null, { metrics: { prometheus: payload } });
  }
}

const populateConfigFromEnv = (config: PluginConfig) => {
  const env = Object.assign({}, process.env);

  if (env['GF_PLUGIN_RENDERING_TIMEZONE']) {
    config.rendering.timezone = env['GF_PLUGIN_RENDERING_TIMEZONE'];
  } else {
    config.rendering.timezone = env['TZ'];
  }

  if (env['GF_PLUGIN_RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR']) {
    config.rendering.maxDeviceScaleFactor = parseFloat(env['GF_PLUGIN_RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR'] as string);
  }

  if (env['GF_PLUGIN_RENDERING_IGNORE_HTTPS_ERRORS']) {
    config.rendering.ignoresHttpsErrors = env['GF_PLUGIN_RENDERING_IGNORE_HTTPS_ERRORS'] === 'true';
  }

  if (env['GF_PLUGIN_RENDERING_CHROME_BIN']) {
    config.rendering.chromeBin = env['GF_PLUGIN_RENDERING_CHROME_BIN'];
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

  if (env['GF_PLUGIN_RENDERING_VERBOSE_LOGGING']) {
    config.rendering.verboseLogging = env['GF_PLUGIN_RENDERING_VERBOSE_LOGGING'] === 'true';
  }

  if (env['GF_PLUGIN_RENDERING_DUMPIO']) {
    config.rendering.dumpio = env['GF_PLUGIN_RENDERING_DUMPIO'] === 'true';
  }

  if (env['GF_PLUGIN_RENDERING_ARGS']) {
    const args = env['GF_PLUGIN_RENDERING_ARGS'] as string;
    if (args.length > 0) {
      const argsList = args.split(',');
      if (argsList.length > 0) {
        config.rendering.args = argsList;
      }
    }
  }
};

interface Metrics {
  up: promClient.Gauge;
  browserInfo: promClient.Gauge;
}

const setupMetrics = (): Metrics => {
  promClient.collectDefaultMetrics();
  const up = new promClient.Gauge({
    name: 'up',
    help: '1 = up, 0 = not up',
  });

  const browserInfo = new promClient.Gauge({
    name: 'grafana_image_renderer_browser_info',
    help: "A metric with a constant '1 value labeled by version of the browser in use",
    labelNames: ['version'],
  });

  return {
    up,
    browserInfo,
  };
};
