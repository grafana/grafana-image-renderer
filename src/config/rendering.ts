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
  pageZoomLevel: number;
  mode: string;
  clustering: ClusteringConfig;
  verboseLogging: boolean;
  dumpio: boolean;
  timingMetrics: boolean;
  headed?: boolean;
  networkConditions?: NetworkConditions;
  emulateNetworkConditions: boolean;
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
};

export function populateRenderingConfigFromEnv(config: RenderingConfig, env: NodeJS.ProcessEnv, isPlugin: boolean) {
  const prefix = isPlugin ? "GF_PLUGIN_" : ""
  const renderingPrefix = isPlugin ? prefix + "RENDERING_" : ""

  if (env[renderingPrefix + 'CHROME_BIN']) {
    config.chromeBin = env[renderingPrefix + 'CHROME_BIN'];
  }

  if (env[prefix + 'RENDERING_ARGS']) {
    const args = env[prefix + 'RENDERING_ARGS'] as string;
    if (args.length > 0) {
      const argsList = args.split(',');
      if (argsList.length > 0) {
        config.args = argsList;
      }
    }
  }

  if (env[renderingPrefix + 'IGNORE_HTTPS_ERRORS']) {
    config.ignoresHttpsErrors = env[renderingPrefix + 'IGNORE_HTTPS_ERRORS'] === 'true';
  }

  if (env[prefix + 'RENDERING_TIMEZONE']) {
    config.timezone = env[prefix + 'RENDERING_TIMEZONE'];
  } else if (env['BROWSER_TZ']) {
    config.timezone = env['BROWSER_TZ'];
  } else if (env['TZ']) {
    config.timezone = env['TZ'];
  }

  if (env[prefix + 'RENDERING_LANGUAGE']) {
    config.acceptLanguage = env[prefix + 'RENDERING_LANGUAGE'];
  }

  if (env[prefix + 'RENDERING_VIEWPORT_WIDTH']) {
    config.width = parseInt(env[prefix + 'RENDERING_VIEWPORT_WIDTH'] as string, 10);
  }

  if (env[prefix + 'RENDERING_VIEWPORT_HEIGHT']) {
    config.height = parseInt(env[prefix + 'RENDERING_VIEWPORT_HEIGHT'] as string, 10);
  }

  if (env[prefix + 'RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR']) {
    config.deviceScaleFactor = parseFloat(env[prefix + 'RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR'] as string);
  }

  if (env[prefix + 'RENDERING_VIEWPORT_MAX_WIDTH']) {
    config.maxWidth = parseInt(env[prefix + 'RENDERING_VIEWPORT_MAX_WIDTH'] as string, 10);
  }

  if (env[prefix + 'RENDERING_VIEWPORT_MAX_HEIGHT']) {
    config.maxHeight = parseInt(env[prefix + 'RENDERING_VIEWPORT_MAX_HEIGHT'] as string, 10);
  }

  if (env[prefix + 'RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR']) {
    config.maxDeviceScaleFactor = parseFloat(env[prefix + 'RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR'] as string);
  }

  if (env[prefix + 'RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL']) {
    config.pageZoomLevel = parseFloat(env[prefix + 'RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL'] as string);
  }

  if (env[prefix + 'RENDERING_MODE']) {
    config.mode = env[prefix + 'RENDERING_MODE'] as string;
  }

  if (env[prefix + 'RENDERING_CLUSTERING_MODE']) {
    config.clustering.mode = env[prefix + 'RENDERING_CLUSTERING_MODE'] as string;
  }

  if (env[prefix + 'RENDERING_CLUSTERING_MAX_CONCURRENCY']) {
    config.clustering.maxConcurrency = parseInt(env[prefix + 'RENDERING_CLUSTERING_MAX_CONCURRENCY'] as string, 10);
  }

  if (env[prefix + 'RENDERING_CLUSTERING_TIMEOUT']) {
    config.clustering.timeout = parseInt(env[prefix + 'RENDERING_CLUSTERING_TIMEOUT'] as string, 10);
  }

  if (env[prefix + 'RENDERING_VERBOSE_LOGGING']) {
    config.verboseLogging = env[prefix + 'RENDERING_VERBOSE_LOGGING'] === 'true';
  }

  if (env[prefix + 'RENDERING_DUMPIO']) {
    config.dumpio = env[prefix + 'RENDERING_DUMPIO'] === 'true';
  }

  if (env[prefix + 'RENDERING_TIMING_METRICS']) {
    config.timingMetrics = env[prefix + 'RENDERING_TIMING_METRICS'] === 'true';
  }
}
