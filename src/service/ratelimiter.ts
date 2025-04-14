import { RateLimiterRedis, RateLimiterMemory } from 'rate-limiter-flexible';
import { Redis } from 'ioredis';

import { RateLimiterConfig } from './config';

export const setupRateLimiter = (config: RateLimiterConfig) => {
  let rateLimiter;

  if (config.redisHost && config.redisPort) {
    const redisClient = new Redis({
      host: config.redisHost || 'localhost',
      port: config.redisPort || 6379,
    });

    rateLimiter = new RateLimiterRedis({
      storeClient: redisClient,
      keyPrefix: 'rate-limit',
      points: config.limitRps, // Maximum number of requests
      duration: 60, // per 60 seconds
    });
  } else {
    // Fallback to in-memory storage
    rateLimiter = new RateLimiterMemory({
      points: config.limitRps,
      duration: 60,
    });
  }

  return rateLimiter;
};
