import * as fs from 'fs';

export interface RenderingConfig {
  chromeBin?: string;
  ignoresHttpsErrors: boolean;
}

export interface ServiceConfig {
  service: {
    port: number;

    metrics: {
      enabled: boolean;
      collectDefaultMetrics: boolean;
      buckets: number[];
    };
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
  ignoresHttpsErrors: false,
};

export const defaultServiceConfig: ServiceConfig = {
  service: {
    port: 8081,
    metrics: {
      enabled: false,
      collectDefaultMetrics: true,
      buckets: [0.5, 1, 3, 5, 7, 10, 20, 30, 60],
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
