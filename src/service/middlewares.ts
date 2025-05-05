import express = require('express');
import * as boom from '@hapi/boom';
import { ImageRenderOptions } from '../types';
import { SecurityConfig, isAuthTokenValid } from '../config/security';
import { RateLimiterAbstract } from 'rate-limiter-flexible';

export const asyncMiddleware = (fn) => (req, res, next) => {
  Promise.resolve(fn(req, res, next)).catch((err) => {
    if (!err.isBoom) {
      return next(boom.badImplementation(err));
    }
    next(err);
  });
};

export const trustedUrlMiddleware = (
  req: express.Request<any, any, any, ImageRenderOptions, any>,
  res: express.Response,
  next: express.NextFunction
) => {
  const queryUrl = req.query.url;

  if (queryUrl && !(queryUrl.startsWith('http://') || queryUrl.startsWith('https://'))) {
    return next(boom.forbidden('Forbidden query url protocol'));
  }

  next();
};

export const authTokenMiddleware = (config: SecurityConfig) => {
  return (req: express.Request<any, any, any, ImageRenderOptions, any>, res: express.Response, next: express.NextFunction) => {
    const headerToken = req.header('X-Auth-Token');
    if (headerToken === undefined || !isAuthTokenValid(config, headerToken)) {
      return next(boom.unauthorized('Unauthorized request'));
    }

    next();
  };
};

export const rateLimiterMiddleware = (rateLimiter: RateLimiterAbstract) => {
  return async (req: express.Request<any, any, any, ImageRenderOptions, any>, res: express.Response, next: express.NextFunction) => {
    const rateLimiterKey = req.header('X-Tenant-ID') || req.ip;
    if (rateLimiterKey === undefined) {
      return next(boom.badRequest('Missing X-Tenant-ID header to use rate limiter'));
    }

    try {
      await rateLimiter.consume(rateLimiterKey);
      next();
    } catch (err) {
      res.set('Retry-After', String(Math.ceil(err.msBeforeNext / 1000)));
      res.status(429).send('Too Many Requests');
    }
  };
};
