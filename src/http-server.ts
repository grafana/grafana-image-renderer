import express = require('express');
import { Logger } from './logger';
import { Browser } from './browser';
import * as boom from 'boom';
import morgan = require('morgan');
import * as promBundle from 'express-prom-bundle';

export class HttpServer {
  app: express.Express;

  constructor(private options, private log: Logger, private browser: Browser) {}

  start() {
    this.app = express();
    this.app.use(morgan('combined'));
    this.registerMetricsMiddleware();
    this.app.get('/', (req: express.Request, res: express.Response) => {
      res.send('Grafana Image Renderer');
    });

    this.app.get('/render', asyncMiddleware(this.render));
    this.app.use((err, req, res, next) => {
      console.error(err);
      return res.status(err.output.statusCode).json(err.output.payload);
    });

    if (this.browser.chromeBin) {
      this.log.info(`Using chromeBin ${this.browser.chromeBin}`);
    }

    if (this.browser.ignoreHTTPSErrors) {
      this.log.info(`Ignoring HTTPS errors`);
    }

    this.app.listen(this.options.port);
    this.log.info(`HTTP Server started, listening on ${this.options.port}`);
  }

  render = async (req: express.Request, res: express.Response) => {
    if (!req.query.url) {
      throw boom.badRequest('Missing url parameter');
    }

    const options = {
      url: req.query.url,
      width: req.query.width,
      height: req.query.height,
      filePath: req.query.filePath,
      timeout: req.query.timeout,
      renderKey: req.query.renderKey,
      domain: req.query.domain,
      timezone: req.query.timezone,
      encoding: req.query.encoding,
    };
    this.log.info(`render request received for ${options.url}`);
    const result = await this.browser.render(options);

    res.sendFile(result.filePath);
  };

  registerMetricsMiddleware() {
    const env = Object.assign({}, process.env);
    let enableMetrics = false;

    if (env['ENABLE_METRICS']) {
      enableMetrics = env['ENABLE_METRICS'] === 'true';
    }

    if (!enableMetrics) {
      return;
    }

    this.log.info('Metrics enabled');

    const bundle = promBundle({
      metricType: 'histogram',
      buckets: [0.5, 1, 3, 5, 7, 10, 20, 30, 60],
      excludeRoutes: [/^((?!(render)).)*$/],
      promClient: {
        collectDefaultMetrics: {},
      },
    } as any);

    this.app.use(bundle);
  }
}

// wrapper for our async route handlers
// probably you want to move it to a new file
const asyncMiddleware = fn => (req, res, next) => {
  Promise.resolve(fn(req, res, next)).catch(err => {
    if (!err.isBoom) {
      return next(boom.badImplementation(err));
    }
    next(err);
  });
};
