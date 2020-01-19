import * as promBundle from 'express-prom-bundle';
import { MetricsConfig } from '../config';
import { Logger } from '../logger';

export const metricsMiddleware = (config: MetricsConfig, log: Logger) => {
  if (!config.enabled) {
    return (req, res, next) => {
      next();
    };
  }

  log.info('Metrics enabled');

  const opts = {
    httpDurationMetricName: 'grafana_image_renderer_service_http_request_duration_seconds',
    metricType: 'histogram',
    buckets: config.requestDurationBuckets,
    excludeRoutes: [/^((?!(render)).)*$/],
    promClient: {},
  } as any;

  if (config.collectDefaultMetrics) {
    opts.promClient.collectDefaultMetrics = {};
  }

  const bundle = promBundle(opts);
  return bundle;
};
