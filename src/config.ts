import * as fs from 'fs';

export interface ClusteringConfig {
  mode: string;
  maxConcurrency: number;
}

export interface RenderingConfig {
  timezone?: string;
  chromeBin?: string;
  ignoresHttpsErrors: boolean;
  mode: string;
  clustering: ClusteringConfig;
}

export interface MetricsConfig {
  enabled: boolean;
  collectDefaultMetrics: boolean;
  requestDurationBuckets: number[];
}

export interface ServiceConfig {
  service: {
    port: number;
    metrics: MetricsConfig;
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
  timezone: undefined,
  chromeBin: undefined,
  ignoresHttpsErrors: false,
  mode: 'default',
  clustering: {
    mode: 'browser',
    maxConcurrency: 5,
  },
};

export const defaultServiceConfig: ServiceConfig = {
  service: {
    port: 8081,
    metrics: {
      enabled: false,
      collectDefaultMetrics: true,
      requestDurationBuckets: [0.5, 1, 3, 5, 7, 10, 20, 30, 60],
    },
  },
  rendering: defaultRenderingConfig,
};

export const defaultPluginConfig: PluginConfig = {
  plugin: {
    grpc: {
      host: '127.0.0.1',
      port: 50059,
    },
  },
  rendering: defaultRenderingConfig,
};

export const readJSONFileSync = (filePath: string): any => {
  const rawdata = fs.readFileSync(filePath, 'utf8');
  return JSON.parse(rawdata);
};
