import * as net from 'net';
import express = require('express');
import { Logger } from '../logger';
import { Browser } from '../browser';
import * as boom from '@hapi/boom';
import morgan = require('morgan');
import { ServiceConfig } from '../config';
import { metricsMiddleware } from './metrics_middleware';
import * as promClient from 'prom-client';
import { RenderOptions } from '../browser/browser';

export class HttpServer {
  app: express.Express;

  constructor(private config: ServiceConfig, private log: Logger, private browser: Browser) {}

  async start() {
    this.app = express();
    this.app.use(
      morgan('combined', {
        skip: (req, res) => {
          return res.statusCode >= 400;
        },
        stream: this.log.debugWriter,
      })
    );
    this.app.use(
      morgan('combined', {
        skip: (req, res) => {
          return res.statusCode < 400;
        },
        stream: this.log.errorWriter,
      })
    );
    this.app.use(metricsMiddleware(this.config.service.metrics, this.log));
    this.app.get('/', (req: express.Request, res: express.Response) => {
      res.send('Grafana Image Renderer');
    });

    this.app.get('/render', asyncMiddleware(this.render));
    this.app.use((err, req, res, next) => {
      if (err.stack) {
        this.log.error('Request failed', 'url', req.url, 'stack', err.stack);
      } else {
        this.log.error('Request failed', 'url', req.url, 'error', err);
      }

      return res.status(err.output.statusCode).json(err.output.payload);
    });

    if (this.config.rendering.chromeBin) {
      this.log.info(`Using chromeBin ${this.config.rendering.chromeBin}`);
    }

    if (this.config.rendering.ignoresHttpsErrors) {
      this.log.info(`Ignoring HTTPS errors`);
    }

    if (this.config.service.host) {
      const server = this.app.listen(this.config.service.port, this.config.service.host, () => {
        const info = server.address() as net.AddressInfo;
        this.log.info(`HTTP Server started, listening at http://${this.config.service.host}:${info.port}`);
      });
    } else {
      const server = this.app.listen(this.config.service.port, () => {
        const info = server.address() as net.AddressInfo;
        this.log.info(`HTTP Server started, listening at http://localhost:${info.port}`);
      });
    }

    const browserInfo = new promClient.Gauge({
      name: 'grafana_image_renderer_browser_info',
      help: "A metric with a constant '1 value labeled by version of the browser in use",
      labelNames: ['version'],
    });

    try {
      const browserVersion = await this.browser.getBrowserVersion();
      browserInfo.labels(browserVersion).set(1);
    } catch {
      this.log.error('Failed to get browser version');
      browserInfo.labels('unknown').set(19);
    }
    await this.browser.start();
  }

  render = async (req: express.Request, res: express.Response) => {
    if (!req.query.url) {
      throw boom.badRequest('Missing url parameter');
    }

    const options: RenderOptions = {
      url: req.query.url,
      width: req.query.width,
      height: req.query.height,
      scale: req.query.scale,
      filePath: req.query.filePath,
      timeout: req.query.timeout,
      renderKey: req.query.renderKey,
      domain: req.query.domain,
      timezone: req.query.timezone,
      encoding: req.query.encoding,
    };
    this.log.debug('Render request received', 'url', options.url);
    req.on('close', err => {
      this.log.debug('Connection closed', 'url', options.url, 'error', err);
    });
    const result = await this.browser.render(options);
    res.sendFile(result.filePath);
  };
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
