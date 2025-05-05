import { defaultRenderingConfig, populateRenderingConfigFromEnv, Mode, RenderingConfig } from '../config/rendering';
import { SecurityConfig } from '../config/security';

export interface MetricsConfig {
  enabled: boolean;
  collectDefaultMetrics: boolean;
  requestDurationBuckets: number[];
}

export interface ConsoleLoggerConfig {
  level?: string;
  json: boolean;
  colorize: boolean;
}

export interface LoggingConfig {
  level: string;
  console?: ConsoleLoggerConfig;
}

export interface RateLimiterConfig {
  enabled: boolean;
  redisHost?: string;
  redisPort?: number;
  requestsPerSecond: number;
}

export interface ServiceConfig {
  service: {
    host?: string;
    port: number;
    protocol?: string;
    certFile?: string;
    certKey?: string;
    minTLSVersion?: string;
    metrics: MetricsConfig;
    logging: LoggingConfig;
    security: SecurityConfig;
    rateLimiter: RateLimiterConfig;
  };
  rendering: RenderingConfig;
}

export const defaultServiceConfig: ServiceConfig = {
  service: {
    host: undefined,
    port: 8081,
    protocol: 'http',
    metrics: {
      enabled: false,
      collectDefaultMetrics: true,
      requestDurationBuckets: [0.5, 1, 3, 5, 7, 10, 20, 30, 60],
    },
    logging: {
      level: 'info',
      console: {
        json: true,
        colorize: false,
      },
    },
    security: {
      authToken: '-',
    },
    rateLimiter: {
      enabled: false,
      requestsPerSecond: 5,
    },
  },
  rendering: defaultRenderingConfig,
};

export function populateServiceConfigFromEnv(config: ServiceConfig, env: NodeJS.ProcessEnv) {
  if (env['HTTP_HOST']) {
    config.service.host = env['HTTP_HOST'];
  }

  if (env['HTTP_PORT']) {
    config.service.port = parseInt(env['HTTP_PORT'] as string, 10);
  }

  if (env['HTTP_PROTOCOL']) {
    config.service.protocol = env['HTTP_PROTOCOL'];
  }

  if (env['HTTP_CERT_FILE']) {
    config.service.certFile = env['HTTP_CERT_FILE'];
  }

  if (env['HTTP_CERT_KEY']) {
    config.service.certKey = env['HTTP_CERT_KEY'];
  }

  if (env['HTTP_MIN_TLS_VERSION']) {
    config.service.minTLSVersion = env['HTTP_MIN_TLS_VERSION'];
  }

  if (env['AUTH_TOKEN']) {
    const authToken = env['AUTH_TOKEN'] as string;
    config.service.security.authToken = authToken.includes(' ') ? authToken.split(' ') : authToken;
  }

  if (env['LOG_LEVEL']) {
    config.service.logging.level = env['LOG_LEVEL'] as string;
  }

  if (env['ENABLE_METRICS']) {
    config.service.metrics.enabled = env['ENABLE_METRICS'] === 'true';
  }

  populateRenderingConfigFromEnv(config.rendering, env, Mode.Server);
}
