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

export interface ServiceConfig {
  service: {
    host?: string;
    port: number;
    metrics: MetricsConfig;
    logging: LoggingConfig;
    security: SecurityConfig;
  };
  rendering: RenderingConfig;
}

export const defaultServiceConfig: ServiceConfig = {
  service: {
    host: undefined,
    port: 8081,
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
