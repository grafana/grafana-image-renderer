import * as fs from 'fs';

export interface ClusteringConfig {
  monitor: boolean;
  mode: string;
  maxConcurrency: number;
  timeout: number;
}

// https://chromedevtools.github.io/devtools-protocol/tot/Network/#method-emulateNetworkConditions
type NetworkConditions = {
  offline: boolean;
  downloadThroughput: number;
  uploadThroughput: number;
  latency: number;
};

export interface RenderingConfig {
  chromeBin?: string;
  args: string[];
  ignoresHttpsErrors: boolean;
  timezone?: string;
  acceptLanguage?: string;
  width: number;
  height: number;
  deviceScaleFactor: number;
  maxWidth: number;
  maxHeight: number;
  maxDeviceScaleFactor: number;
  mode: string;
  clustering: ClusteringConfig;
  verboseLogging: boolean;
  dumpio: boolean;
  timingMetrics: boolean;
  headed?: boolean;
  networkConditions?: NetworkConditions;
  emulateNetworkConditions: boolean;
}

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
  };
  rendering: RenderingConfig;
}

export interface PluginConfig {
  plugin: {
    grpc: {
      host: string;
      port: number;
    };
  };
  rendering: RenderingConfig;
}

const defaultRenderingConfig: RenderingConfig = {
  chromeBin: undefined,
  args: ['--no-sandbox', '--disable-gpu'],
  ignoresHttpsErrors: false,
  timezone: undefined,
  acceptLanguage: undefined,
  width: 1000,
  height: 500,
  headed: false,
  deviceScaleFactor: 1,
  maxWidth: 3000,
  maxHeight: 3000,
  maxDeviceScaleFactor: 4,
  mode: 'default',
  clustering: {
    monitor: false,
    mode: 'browser',
    maxConcurrency: 5,
    timeout: 30,
  },
  emulateNetworkConditions: false,
  verboseLogging: false,
  dumpio: false,
  timingMetrics: false,
};

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
  },
  rendering: defaultRenderingConfig,
};

export const defaultPluginConfig: PluginConfig = {
  plugin: {
    grpc: {
      host: '127.0.0.1',
      port: 0,
    },
  },
  rendering: defaultRenderingConfig,
};

export const readJSONFileSync = (filePath: string): any => {
  const rawdata = fs.readFileSync(filePath, 'utf8');
  return JSON.parse(rawdata);
};
