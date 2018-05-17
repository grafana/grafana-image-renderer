import * as fs from 'fs';
import express = require('express');
import { Logger } from './logger';
import { Browser } from './browser';
import * as boom from 'boom';

export class HttpServer {
  app: express.Express;

  constructor(private options,
              private log: Logger,
              private browser: Browser) {
  }

  start() {
    this.app = express();

    this.app.get('/', (req: express.Request, res: express.Response) => {
      res.send('Grafana Image Renderer');
    });

    this.app.get('/render', asyncMiddleware(this.render));

    this.app.listen(this.options.port);
    this.log.info(`HTTP Server started, listening on ${this.options.port}`);
  }

  render = async (req: express.Request, res: express.Response) => {
    let result = await this.browser.render();

    res.sendFile(result.imagePath);

    this.log.info(req.query);
  }
}

// wrapper for our async route handlers
// probably you want to move it to a new file
const asyncMiddleware = fn => (req, res, next) => {
  Promise.resolve(fn(req, res, next)).catch((err) => {
    if (!err.isBoom) {
      return next(boom.badImplementation(err));
    }
    next(err);
  });
};

const readFile = (path, opts = 'utf8') => {
  return new Promise((res, rej) => {
    fs.readFile(path, opts, (err, data) => {
      if (err) {
        rej(err);
      } else {
        res(data);
      }
    });
  });
};
