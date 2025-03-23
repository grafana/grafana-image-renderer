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

export interface TracesConfig {
  url: string;
  serviceName?: string;
}

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
  pageZoomLevel: number;
  mode: string;
  clustering: ClusteringConfig;
  verboseLogging: boolean;
  dumpio: boolean;
  timingMetrics: boolean;
  headed?: boolean;
  networkConditions?: NetworkConditions;
  emulateNetworkConditions: boolean;
  tracing: TracesConfig;
}

export const defaultRenderingConfig: RenderingConfig = {
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
  pageZoomLevel: 1,
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
  tracing: {
    url: '',
    serviceName: '',
  },
};

export enum Mode {
  Plugin = 'plugin',
  Server = 'server',
}

type Keys<T> = {
  [K in keyof T]?: T[K] extends object ? (T[K] extends any[] ? string : Keys<T[K]>) : string;
};

const envConfig: Record<Mode, Keys<RenderingConfig>> = {
  server: {
    chromeBin: 'CHROME_BIN',
    args: 'RENDERING_ARGS',
    ignoresHttpsErrors: 'IGNORE_HTTPS_ERRORS',
    timezone: 'BROWSER_TZ',
    acceptLanguage: 'RENDERING_LANGUAGE',
    width: 'RENDERING_VIEWPORT_WIDTH',
    height: 'RENDERING_VIEWPORT_HEIGHT',
    deviceScaleFactor: 'RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR',
    maxWidth: 'RENDERING_VIEWPORT_MAX_WIDTH',
    maxHeight: 'RENDERING_VIEWPORT_MAX_HEIGHT',
    maxDeviceScaleFactor: 'RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR',
    pageZoomLevel: 'RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL',
    mode: 'RENDERING_MODE',
    clustering: {
      mode: 'RENDERING_CLUSTERING_MODE',
      maxConcurrency: 'RENDERING_CLUSTERING_MAX_CONCURRENCY',
      timeout: 'RENDERING_CLUSTERING_TIMEOUT',
    },
    verboseLogging: 'RENDERING_VERBOSE_LOGGING',
    dumpio: 'RENDERING_DUMPIO',
    timingMetrics: 'RENDERING_TIMING_METRICS',
    tracing: {
      url: 'RENDERING_TRACING_URL',
    },
  },
  plugin: {
    chromeBin: 'GF_PLUGIN_RENDERING_CHROME_BIN',
    args: 'GF_PLUGIN_RENDERING_ARGS',
    ignoresHttpsErrors: 'GF_PLUGIN_RENDERING_IGNORE_HTTPS_ERRORS',
    timezone: 'GF_PLUGIN_RENDERING_TIMEZONE',
    acceptLanguage: 'GF_PLUGIN_RENDERING_LANGUAGE',
    width: 'GF_PLUGIN_RENDERING_VIEWPORT_WIDTH',
    height: 'GF_PLUGIN_RENDERING_VIEWPORT_HEIGHT',
    deviceScaleFactor: 'GF_PLUGIN_RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR',
    maxWidth: 'GF_PLUGIN_RENDERING_VIEWPORT_MAX_WIDTH',
    maxHeight: 'GF_PLUGIN_RENDERING_VIEWPORT_MAX_HEIGHT',
    maxDeviceScaleFactor: 'GF_PLUGIN_RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR',
    pageZoomLevel: 'GF_PLUGIN_RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL',
    mode: 'GF_PLUGIN_RENDERING_MODE',
    clustering: {
      mode: 'GF_PLUGIN_RENDERING_CLUSTERING_MODE',
      maxConcurrency: 'GF_PLUGIN_RENDERING_CLUSTERING_MAX_CONCURRENCY',
      timeout: 'GF_PLUGIN_RENDERING_CLUSTERING_TIMEOUT',
    },
    verboseLogging: 'GF_PLUGIN_RENDERING_VERBOSE_LOGGING',
    dumpio: 'GF_PLUGIN_RENDERING_DUMPIO',
    timingMetrics: 'GF_PLUGIN_RENDERING_TIMING_METRICS',
    tracing: {
      url: 'GF_PLUGIN_RENDERING_TRACING_URL',

    },
  },
};

export function populateRenderingConfigFromEnv(config: RenderingConfig, env: NodeJS.ProcessEnv, mode: Mode) {
  const envKeys = envConfig[mode];

  if (env[envKeys.chromeBin!]) {
    config.chromeBin = env[envKeys.chromeBin!];
  }

  if (env[envKeys.args!]) {
    const args = env[envKeys.args!] as string;
    if (args.length > 0) {
      const argsList = args.split(',');
      if (argsList.length > 0) {
        config.args = argsList;
      }
    }
  }

  if (env[envKeys.ignoresHttpsErrors!]) {
    config.ignoresHttpsErrors = env[envKeys.ignoresHttpsErrors!] === 'true';
  }

  if (env[envKeys.timezone!]) {
    config.timezone = env[envKeys.timezone!];
  } else if (env['TZ']) {
    config.timezone = env['TZ'];
  }

  if (env[envKeys.acceptLanguage!]) {
    config.acceptLanguage = env[envKeys.acceptLanguage!];
  }

  if (env[envKeys.width!]) {
    config.width = parseInt(env[envKeys.width!] as string, 10);
  }

  if (env[envKeys.height!]) {
    config.height = parseInt(env[envKeys.height!] as string, 10);
  }

  if (env[envKeys.deviceScaleFactor!]) {
    config.deviceScaleFactor = parseFloat(env[envKeys.deviceScaleFactor!] as string);
  }

  if (env[envKeys.maxWidth!]) {
    config.maxWidth = parseInt(env[envKeys.maxWidth!] as string, 10);
  }

  if (env[envKeys.maxHeight!]) {
    config.maxHeight = parseInt(env[envKeys.maxHeight!] as string, 10);
  }

  if (env[envKeys.maxDeviceScaleFactor!]) {
    config.maxDeviceScaleFactor = parseFloat(env[envKeys.maxDeviceScaleFactor!] as string);
  }

  if (env[envKeys.pageZoomLevel!]) {
    config.pageZoomLevel = parseFloat(env[envKeys.pageZoomLevel!] as string);
  }

  if (env[envKeys.mode!]) {
    config.mode = env[envKeys.mode!] as string;
  }

  if (env[envKeys.clustering?.mode!]) {
    config.clustering.mode = env[envKeys.clustering?.mode!] as string;
  }

  if (env[envKeys.clustering?.maxConcurrency!]) {
    config.clustering.maxConcurrency = parseInt(env[envKeys.clustering?.maxConcurrency!] as string, 10);
  }

  if (env[envKeys.clustering?.timeout!]) {
    config.clustering.timeout = parseInt(env[envKeys.clustering?.timeout!] as string, 10);
  }

  if (env[envKeys.verboseLogging!]) {
    config.verboseLogging = env[envKeys.verboseLogging!] === 'true';
  }

  if (env[envKeys.dumpio!]) {
    config.dumpio = env[envKeys.dumpio!] === 'true';
  }

  if (env[envKeys.timingMetrics!]) {
    config.timingMetrics = env[envKeys.timingMetrics!] === 'true';
  }

  if (env[envKeys.tracing?.url!]) {
    config.tracing.url = env[envKeys.tracing?.url!] as string;
  }
}
