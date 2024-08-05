import express = require('express');
import * as boom from '@hapi/boom';
import { ImageRenderOptions } from '../types';
import { SecurityConfig, isAuthTokenValid } from '../config/security';
import {context, propagation, trace} from '@opentelemetry/api';
import {defaultServiceConfig, ServiceConfig} from "./config";
import {ConsoleLogger} from "../logger";

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
    const extractedContext = propagation.extract(context.active(), req.headers);
    let config2: ServiceConfig = defaultServiceConfig;

    const logger = new ConsoleLogger(config2.service.logging);

    // Check if traceparent header is present
    const traceparent = req.header('traceparent');
    logger.debug('Traceparent header: ', 'traceparent', traceparent);

    if (headerToken === undefined || !isAuthTokenValid(config, headerToken)) {
      return next(boom.unauthorized('Unauthorized request'));
    }

    context.with(extractedContext, () => {
      const currentSpan = trace.getSpan(context.active());
      logger.debug('Current span after context extraction: ', 'val', currentSpan ? currentSpan.spanContext().traceId : 'No active span');

      next();
    });
  };
};
