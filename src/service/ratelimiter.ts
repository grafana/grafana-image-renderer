import { RateLimiterRedis, RateLimiterMemory } from 'rate-limiter-flexible';
import { Redis } from 'ioredis';

import { RateLimiterConfig } from './config';
import { Logger } from '../logger';

export const setupRateLimiter = (config: RateLimiterConfig, log: Logger) => {
  let rateLimiter;

  if (config.redisHost && config.redisPort) {
    const redisClient = new Redis({
      host: config.redisHost,
      port: config.redisPort,
    });

    rateLimiter = new RateLimiterRedis({
      storeClient: redisClient,
      keyPrefix: 'rate-limit',
      points: config.requestsPerSecond, // Maximum number of requests
      duration: 1, // per second
    });

    log.info('Rate limiter enabled using Redis', 'requestsPerSecond', config.requestsPerSecond);
  } else {
    // Fallback to in-memory storage
    rateLimiter = new RateLimiterMemory({
      points: config.requestsPerSecond,
      duration: 1,
    });

    log.info('Rate limiter enabled using in-memory storage', 'requestsPerSecond', config.requestsPerSecond);
  }

  return rateLimiter;
};
