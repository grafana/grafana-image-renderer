import * as promBundle from 'express-prom-bundle';
import * as promClient from 'prom-client';
import * as onFinished from 'on-finished';
import express = require('express');

import { MetricsConfig } from './config';
import { Logger } from '../logger';

export const setupHttpServerMetrics = (app: express.Express, config: MetricsConfig, log: Logger) => {
  log.info(
    'Metrics enabled',
    'collectDefaultMetrics',
    config.collectDefaultMetrics,
    'requestDurationBuckets',
    config.requestDurationBuckets.join(',')
  );

  // Exclude all non-rendering endpoints:
  //   - endpoints that do not include render
  //   - /render/version
  const excludeRegExp = /^(((?!(render)).)*|.*version.*)$/;

  const opts = {
    httpDurationMetricName: 'grafana_image_renderer_service_http_request_duration_seconds',
    metricType: 'histogram',
    buckets: config.requestDurationBuckets,
    excludeRoutes: [excludeRegExp],
    promClient: {},
    formatStatusCode: (res) => {
      if (res && res.req && res.req.aborted) {
        // Nginx non-standard code 499 Client Closed Request
        // Used when the client has closed the request before
        // the server could send a response.
        return 499;
      }

      return res.status_code || res.statusCode;
    },
  } as any;

  if (config.collectDefaultMetrics) {
    opts.promClient.collectDefaultMetrics = {};
  }

  const metricsMiddleware = promBundle(opts);
  app.use(metricsMiddleware);

  const httpRequestsInFlight = new promClient.Gauge({
    name: 'grafana_image_renderer_http_request_in_flight',
    help: 'A gauge of requests currently being served by the image renderer.',
  });
  app.use(requestsInFlightMiddleware(httpRequestsInFlight, excludeRegExp));
};

const requestsInFlightMiddleware = (httpRequestsInFlight: promClient.Gauge, excludeRegExp: RegExp) => {
  return (req, res, next) => {
    const path = req.originalUrl || req.url;
    if (path.match(excludeRegExp)) {
      return next();
    }

    httpRequestsInFlight.inc();
    onFinished(res, () => {
      httpRequestsInFlight.dec();
    });

    next();
  };
};
