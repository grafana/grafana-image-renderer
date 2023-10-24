import * as fs from 'fs';
import * as path from 'path';
import * as net from 'net';
import express = require('express');
import * as boom from '@hapi/boom';
import morgan = require('morgan');
import * as promClient from 'prom-client';
import { Logger } from '../logger';
import { Browser, createBrowser } from '../browser';
import { ServiceConfig } from './config';
import { setupHttpServerMetrics } from './metrics';
import { HTTPHeaders, ImageRenderOptions, RenderOptions } from '../types';
import { Sanitizer } from '../sanitizer/Sanitizer';
import * as bodyParser from 'body-parser';
import * as multer from 'multer';
import { isSanitizeRequest } from '../sanitizer/types';
import * as contentDisposition from 'content-disposition';
import { asyncMiddleware, trustedUrlMiddleware, authTokenMiddleware } from './middlewares';

const upload = multer({ storage: multer.memoryStorage() });

enum SanitizeRequestPartName {
  'file' = 'file',
  'config' = 'config',
}

export class HttpServer {
  app: express.Express;
  browser: Browser;

  constructor(private config: ServiceConfig, private log: Logger, private sanitizer: Sanitizer) { }

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

    this.app.use(bodyParser.json());

    if (this.config.service.metrics.enabled) {
      setupHttpServerMetrics(this.app, this.config.service.metrics, this.log);
    }

    this.app.get('/', (req: express.Request, res: express.Response) => {
      res.send('Grafana Image Renderer');
    });

    // Middlewares for /render endpoints
    this.app.use('/render', authTokenMiddleware(this.config.service.security), trustedUrlMiddleware);

    // Set up /render endpoints
    this.app.get('/render', asyncMiddleware(this.render));
    this.app.get('/render/csv', asyncMiddleware(this.renderCSV));
    this.app.get('/render/version', (req: express.Request, res: express.Response) => {
      const pluginInfo = require('../../plugin.json');
      res.send({ version: pluginInfo.info.version });
    });

    // Middlewares for /sanitize endpoints
    this.app.use('/sanitize', authTokenMiddleware(this.config.service.security));

    // Set up /sanitize endpoints
    this.app.post(
      '/sanitize',
      upload.fields([
        { name: SanitizeRequestPartName.file, maxCount: 1 },
        { name: SanitizeRequestPartName.config, maxCount: 1 },
      ]),
      asyncMiddleware(this.sanitize)
    );

    this.app.use((err, req, res, next) => {
      if (err.stack) {
        this.log.error('Request failed', 'url', req.url, 'stack', err.stack);
      } else {
        this.log.error('Request failed', 'url', req.url, 'error', err);
      }

      if (err.output) {
        return res.status(err.output.statusCode).json(err.output.payload);
      }

      return res.status(500).json(err);
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

  sanitize = async (req: express.Request<any, { error: string }>, res: express.Response<{ error: string }>) => {
    const file = req.files?.[SanitizeRequestPartName.file]?.[0] as Express.Multer.File | undefined;
    if (!file) {
      throw boom.badRequest('missing file');
    }

    const configFile = req.files?.[SanitizeRequestPartName.config]?.[0] as Express.Multer.File | undefined;
    if (!configFile) {
      throw boom.badRequest('missing config');
    }

    const config = JSON.parse(configFile.buffer.toString());

    const sanitizeReq = {
      ...config,
      content: file.buffer,
    };

    if (!isSanitizeRequest(sanitizeReq)) {
      throw boom.badRequest('invalid request: ' + JSON.stringify(config));
    }

    this.log.debug('Sanitize request received', 'contentLength', file.size, 'name', file.filename, 'config', JSON.stringify(config));

    try {
      const sanitizeResponse = this.sanitizer.sanitize(sanitizeReq);
      res.writeHead(200, {
        'Content-Disposition': `attachment;filename=${file.filename ?? 'sanitized'}`,
        'Content-Length': sanitizeResponse.sanitized.length,
        'Content-Type': file.mimetype ?? 'application/octet-stream',
      });
      return res.end(sanitizeResponse.sanitized);
    } catch (e) {
      this.log.error('Sanitization failed', 'filesize', file.size, 'name', file.filename, 'error', e.stack);
      return res.status(500).json({ error: e.message });
    }
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
      res.setHeader('Content-Disposition', contentDisposition(result.fileName));
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
