import * as fs from 'fs';
import * as path from 'path';
import * as net from 'net';
import express = require('express');
import * as boom from '@hapi/boom';
import morgan = require('morgan');
import * as promClient from 'prom-client';
import { Logger } from '../logger';
import { Browser, createBrowser } from '../browser';
import { ServiceConfig } from '../config';
import { setupHttpServerMetrics } from './metrics_middleware';
import { HTTPHeaders, ImageRenderOptions, RenderOptions } from '../types';

export class HttpServer {
  app: express.Express;
  browser: Browser;

  constructor(private config: ServiceConfig, private log: Logger) {}

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

    if (this.config.service.metrics.enabled) {
      setupHttpServerMetrics(this.app, this.config.service.metrics, this.log);
    }
    this.app.get('/', (req: express.Request, res: express.Response) => {
      res.send('Grafana Image Renderer');
    });
    this.app.get('/render/version', (req: express.Request, res: express.Response) => {
      const pluginInfo = require('../../plugin.json');
      res.send({ version: pluginInfo.info.version });
    });

    this.app.get('/render', asyncMiddleware(this.render));
    this.app.get('/render/csv', asyncMiddleware(this.renderCSV));
    this.app.use((err, req, res, next) => {
      if (err.stack) {
        this.log.error('Request failed', 'url', req.url, 'stack', err.stack);
      } else {
        this.log.error('Request failed', 'url', req.url, 'error', err);
      }

      return res.status(err.output.statusCode).json(err.output.payload);
    });

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

    const metrics = {
      durationHistogram: new promClient.Histogram({
        name: 'grafana_image_renderer_step_duration_seconds',
        help: 'duration histogram of browser steps for rendering an image labeled with: step',
        labelNames: ['step'],
        buckets: [0.1, 0.3, 0.5, 1, 3, 5, 10, 20, 30],
      }),
    };
    this.browser = createBrowser(this.config.rendering, this.log, metrics);

    if (this.config.service.metrics.enabled) {
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
        browserInfo.labels('unknown').set(0);
      }
    }

    await this.browser.start();
  }

  render = async (req: express.Request<any, any, any, ImageRenderOptions, any>, res: express.Response, next: express.NextFunction) => {
    if (!req.query.url) {
      throw boom.badRequest('Missing url parameter');
    }

    const headers: HTTPHeaders = {};

    if (req.headers['Accept-Language']) {
      headers['Accept-Language'] = (req.headers['Accept-Language'] as string[]).join(';');
    }

    const options: ImageRenderOptions = {
      url: req.query.url,
      width: req.query.width,
      height: req.query.height,
      filePath: req.query.filePath,
      timeout: req.query.timeout,
      renderKey: req.query.renderKey,
      domain: req.query.domain,
      timezone: req.query.timezone,
      encoding: req.query.encoding,
      deviceScaleFactor: req.query.deviceScaleFactor,
      headers: headers,
    };

    this.log.debug('Render request received', 'url', options.url);
    req.on('close', (err) => {
      this.log.debug('Connection closed', 'url', options.url, 'error', err);
    });

    const result = await this.browser.render(options);

    res.sendFile(result.filePath, (err) => {
      if (err) {
        next(err);
      } else {
        try {
          this.log.debug('Deleting temporary file', 'file', result.filePath);
          fs.unlinkSync(result.filePath);
        } catch (e) {
          this.log.error('Failed to delete temporary file', 'file', result.filePath);
        }
      }
    });
  };

  renderCSV = async (req: express.Request<any, any, any, RenderOptions, any>, res: express.Response, next: express.NextFunction) => {
    if (!req.query.url) {
      throw boom.badRequest('Missing url parameter');
    }

    const headers: HTTPHeaders = {};

    if (req.headers['Accept-Language']) {
      headers['Accept-Language'] = (req.headers['Accept-Language'] as string[]).join(';');
    }

    const options: RenderOptions = {
      url: req.query.url,
      filePath: req.query.filePath,
      timeout: req.query.timeout,
      renderKey: req.query.renderKey,
      domain: req.query.domain,
      timezone: req.query.timezone,
      encoding: req.query.encoding,
      headers: headers,
    };

    this.log.debug('Render request received', 'url', options.url);
    req.on('close', (err) => {
      this.log.debug('Connection closed', 'url', options.url, 'error', err);
    });
    const result = await this.browser.renderCSV(options);

    if (result.fileName) {
      res.setHeader('Content-Disposition', `attachment; filename="${result.fileName}"`);
    }
    res.sendFile(result.filePath, (err) => {
      if (err) {
        next(err);
      } else {
        try {
          this.log.debug('Deleting temporary file', 'file', result.filePath);
          fs.unlinkSync(result.filePath);
          if (!options.filePath) {
            fs.rmdirSync(path.dirname(result.filePath));
          }
        } catch (e) {
          this.log.error('Failed to delete temporary file', 'file', result.filePath);
        }
      }
    });
  };
}

// wrapper for our async route handlers
// probably you want to move it to a new file
const asyncMiddleware = (fn) => (req, res, next) => {
  Promise.resolve(fn(req, res, next)).catch((err) => {
    if (!err.isBoom) {
      return next(boom.badImplementation(err));
    }
    next(err);
  });
};
