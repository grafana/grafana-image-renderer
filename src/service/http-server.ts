import bodyParser from 'body-parser';
import * as boom from '@hapi/boom';
import contentDisposition from 'content-disposition';
import express, { Express, Request, Response, NextFunction } from 'express';
import fs from 'fs';
import http from 'http';
import https from 'https';
import morgan from 'morgan';
import multer from 'multer';
import net from 'net';
import path from 'path';
import os from 'os';
import * as promClient from 'prom-client';

import { Logger } from '../logger';
import { Browser, createBrowser } from '../browser';
import { ServiceConfig } from './config';
import { setupHttpServerMetrics } from './metrics';
import { setupRateLimiter } from './ratelimiter';
import { HTTPHeaders, ImageRenderOptions, RenderOptions } from '../types';
import { Sanitizer } from '../sanitizer/Sanitizer';
import { isSanitizeRequest } from '../sanitizer/types';
import { asyncMiddleware, trustedUrlMiddleware, authTokenMiddleware, rateLimiterMiddleware } from './middlewares';
import { SecureVersion } from 'tls';

import pluginInfo from '../../plugin.json';

const upload = multer({ storage: multer.memoryStorage() });

enum SanitizeRequestPartName {
  'file' = 'file',
  'config' = 'config',
}

export class HttpServer {
  app: Express;
  browser: Browser;
  server: http.Server;

  constructor(private config: ServiceConfig, private log: Logger, private sanitizer: Sanitizer) {}

  async start() {
    this.app = express();

    this.app.use(
      morgan('combined', {
        skip: (_, res) => {
          return res.statusCode >= 400;
        },
        stream: this.log.debugWriter,
      } as morgan.Options<Request, Response>)
    );
    this.app.use(
      morgan('combined', {
        skip: (_, res) => {
          return res.statusCode < 400;
        },
        stream: this.log.errorWriter,
      } as morgan.Options<Request, Response>)
    );

    this.app.use(bodyParser.json());

    if (this.config.service.metrics.enabled) {
      setupHttpServerMetrics(this.app, this.config.service.metrics, this.log);
    }

    this.app.get('/', (_: Request, res: Response) => {
      res.send('Grafana Image Renderer');
    });

    // Middlewares for /render endpoints
    this.app.use('/render', authTokenMiddleware(this.config.service.security), trustedUrlMiddleware);
    const rateLimiterConfig = this.config.service.rateLimiter;
    if (rateLimiterConfig.enabled) {
      const rateLimiter = setupRateLimiter(rateLimiterConfig, this.log);
      this.app.use('/render', rateLimiterMiddleware(rateLimiter));
    }

    // Set up /render endpoints
    this.app.get('/render', asyncMiddleware(this.render));
    this.app.get('/render/csv', asyncMiddleware(this.renderCSV));
    this.app.get('/render/version', (_: Request, res: Response) => {
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

    this.app.use((err: Error | boom.Boom<any>, req: Request, res: Response, _: NextFunction) => {
      if (err.stack) {
        this.log.error('Request failed', 'url', req.url, 'stack', err.stack);
      } else {
        this.log.error('Request failed', 'url', req.url, 'error', err);
      }

      if (boom.isBoom(err) && err.output) {
        return res.status(err.output.statusCode).json(err.output.payload);
      }

      return res.status(500).json(err);
    });

    this.createServer();

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
        this.log.info('Browser version', 'version', browserVersion);
        browserInfo.labels(browserVersion).set(1);
      } catch {
        this.log.error('Failed to get browser version');
        browserInfo.labels('unknown').set(0);
      }
    }

    await this.browser.start();
  }

  createServer() {
    const { protocol, host, port } = this.config.service;
    if (protocol === 'https') {
      const { certFile, certKey, minTLSVersion } = this.config.service;
      if (!certFile || !certKey) {
        throw new Error('No cert file or cert key provided, cannot start HTTPS server');
      }

      if (minTLSVersion && minTLSVersion !== 'TLSv1.2' && minTLSVersion !== 'TLSv1.3') {
        throw new Error('Only allowed TLS min versions are TLSv1.2 and TLSv1.3');
      }

      const options = {
        cert: fs.readFileSync(certFile),
        key: fs.readFileSync(certKey),

        maxVersion: 'TLSv1.3' as SecureVersion,
        minVersion: (minTLSVersion || 'TLSv1.2') as SecureVersion,
      };

      this.server = https.createServer(options, this.app);
    } else {
      this.server = http.createServer(this.app);
    }

    if (host) {
      this.server.listen(port, host, () => {
        const info = this.server.address() as net.AddressInfo;
        this.log.info(`${protocol?.toUpperCase()} Server started, listening at ${protocol}://${host}:${info.port}`);
      });
    } else {
      this.server.listen(port, () => {
        const info = this.server.address() as net.AddressInfo;
        this.log.info(`${protocol?.toUpperCase()} Server started, listening at ${protocol}://localhost:${info.port}`);
      });
    }
  }

  close() {
    this.server.close();
  }

  render = async (req: express.Request<any, any, any, ImageRenderOptions, any>, res: express.Response, next: express.NextFunction) => {
    const abortController = new AbortController();
    const { signal } = abortController;

    if (!req.query.url) {
      throw boom.badRequest('Missing url parameter');
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
      headers: this.getHeaders(req),
    };

    const xdgTempDir = await this.createTempXdgDir();
    if (xdgTempDir) {
      options.extraEnv = {
        XDG_CACHE_HOME: process.env?.['XDG_CACHE_HOME'] || xdgTempDir,
        XDG_CONFIG_HOME: process.env?.['XDG_CONFIG_HOME'] || xdgTempDir,
      };
    }

    this.log.debug('Render request received', 'url', options.url);
    req.on('close', (err) => {
      this.log.debug('Connection closed', 'url', options.url, 'error', err);
      this.removeTempXdgDir(xdgTempDir);
      abortController.abort();
    });

    try {
      const result = await this.browser.render(options, signal);

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
    } catch (e) {
      this.log.error('Render failed', 'url', options.url, 'error', e.stack);
      return res.status(500).json({ error: e.message });
    }
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
    const abortController = new AbortController();
    const { signal } = abortController;

    if (!req.query.url) {
      throw boom.badRequest('Missing url parameter');
    }

    const options: RenderOptions = {
      url: req.query.url,
      filePath: req.query.filePath,
      timeout: req.query.timeout,
      renderKey: req.query.renderKey,
      domain: req.query.domain,
      timezone: req.query.timezone,
      encoding: req.query.encoding,
      headers: this.getHeaders(req),
    };

    const xdgTempDir = await this.createTempXdgDir();
    if (xdgTempDir) {
      options.extraEnv = {
        XDG_CACHE_HOME: process.env?.['XDG_CACHE_HOME'] || xdgTempDir,
        XDG_CONFIG_HOME: process.env?.['XDG_CONFIG_HOME'] || xdgTempDir,
      };
    }

    this.log.debug('Render request received', 'url', options.url);
    req.on('close', (err) => {
      this.log.debug('Connection closed', 'url', options.url, 'error', err);
      this.removeTempXdgDir(xdgTempDir);
      abortController.abort();
    });

    try {
      const result = await this.browser.renderCSV(options, signal);

      if (result.fileName) {
        res.setHeader('Content-Disposition', contentDisposition(result.fileName));
      }
      res.sendFile(result.filePath, (err) => {
        if (err) {
          next(err);
        } else {
          try {
            this.log.debug('Deleting temporary file', 'file', result.filePath);
            fs.unlink(result.filePath, (err) => {
              if (err) {
                throw err;
              }

              if (!options.filePath) {
                fs.rmdir(path.dirname(result.filePath), () => {});
              }
            });
          } catch (e) {
            this.log.error('Failed to delete temporary file', 'file', result.filePath, 'error', e.message);
          }
        }
      });
    } catch (e) {
      this.log.error('Render CSV failed', 'url', options.url, 'error', e.stack);
      return res.status(500).json({ error: e.message });
    }
  };

  getHeaders(req: express.Request<any, any, any, RenderOptions, any>): HTTPHeaders {
    const headers: HTTPHeaders = {};

    if (req.headers['Accept-Language']) {
      headers['Accept-Language'] = (req.headers['Accept-Language'] as string[]).join(';');
    }

    // Propagate traces (only if tracing is enabled)
    if (this.config.rendering.tracing.url && req.headers['traceparent']) {
      headers['traceparent'] = req.headers['traceparent'] as string;
      headers['tracestate'] = (req.headers['tracestate'] as string) ?? '';
    }

    return headers;
  }

  async createTempXdgDir(): Promise<string | null> {
    // If both are set, we can skip creating the temp dir.
    if (process.env?.['XDG_CACHE_HOME'] && process.env?.['XDG_CONFIG_HOME']) {
      return null;
    }

    let xdgTempDir: string | null = null;

    try {
      xdgTempDir = await fs.promises.mkdtemp(path.join(os.tmpdir(), 'xdg-'));
      if (!xdgTempDir) {
        throw new Error('fs.promises.mkdtemp returned a null dir');
      }

      this.log.debug('Created temporary XDG directory', 'dir', xdgTempDir);
    } catch (e) {
      this.log.error('Failed to create temporary XDG directory', 'error', e.message);
    }

    return xdgTempDir;
  }

  removeTempXdgDir(xdgTempDir: string | null) {
    if (xdgTempDir) {
      fs.rm(xdgTempDir, { recursive: true, maxRetries: 3 }, (err) => {
        if (err) {
          this.log.error('Failed to delete temporary XDG directory', 'dir', xdgTempDir, 'error', err.message);
        } else {
          this.log.debug('Deleted temporary XDG directory', 'dir', xdgTempDir);
        }
      });
    }
  }
}
