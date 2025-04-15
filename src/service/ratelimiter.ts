import { RateLimiterRedis, RateLimiterMemory } from 'rate-limiter-flexible';
import { Redis } from 'ioredis';

import { RateLimiterConfig } from './config';
import { Logger } from '../logger';

export const setupRateLimiter = (config: RateLimiterConfig, log: Logger) => {
  let rateLimiter;

  if (config.redisHost && config.redisPort) {
    const redisClient = new Redis({
      host: config.redisHost || 'localhost',
      port: config.redisPort || 6379,
    });

    rateLimiter = new RateLimiterRedis({
      storeClient: redisClient,
      keyPrefix: 'rate-limit',
      points: config.requestsPerMinute, // Maximum number of requests
      duration: 60, // per 60 seconds
    });

    log.info('Rate limiter enabled using Redis', 'requestsPerMinute', config.requestsPerMinute);
  } else {
    // Fallback to in-memory storage
    rateLimiter = new RateLimiterMemory({
      points: config.requestsPerMinute,
      duration: 60,
    });

    log.info('Rate limiter enabled using in-memory storage', 'requestsPerMinute', config.requestsPerMinute);
  }

  return rateLimiter;
};
