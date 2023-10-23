import * as path from 'path';
import * as _ from 'lodash';
import * as fs from 'fs';
import { Browser, computeExecutablePath } from '@puppeteer/browsers';
import { RenderGRPCPluginV2 } from './plugin/v2/grpc_plugin';
import { HttpServer } from './service/http-server';
import { ConsoleLogger, PluginLogger } from './logger';
import * as minimist from 'minimist';
import { defaultPluginConfig, defaultServiceConfig, readJSONFileSync, PluginConfig, ServiceConfig, RenderingConfig } from './config';
import { serve } from './node-plugin';
import { createSanitizer } from './sanitizer/Sanitizer';

async function main() {
  const argv = minimist(process.argv.slice(2));
  const env = Object.assign({}, process.env);
  const command = argv._[0];

  // See https://github.com/grafana/grafana-image-renderer/issues/460
  process.env["PUPPETEER_DISABLE_HEADLESS_WARNING"] = "true"

  if (command === undefined) {
    const logger = new PluginLogger();
    const config: PluginConfig = defaultPluginConfig;
    populatePluginConfigFromEnv(config, env);
    if (!config.rendering.chromeBin && (process as any).pkg) {
      const execPath = path.dirname(process.execPath);
      const chromeInfoFile = fs.readFileSync(path.resolve(execPath, 'chrome-info.json'), 'utf8');
      const chromeInfo = JSON.parse(chromeInfoFile);

      config.rendering.chromeBin = computeExecutablePath({
        cacheDir: path.dirname(process.execPath),
        browser: Browser.CHROME,
        buildId: chromeInfo.buildId,
      });
      logger.debug(`Setting chromeBin to ${config.rendering.chromeBin}`);
    }

    await serve({
      handshakeConfig: {
        protocolVersion: 2,
        magicCookieKey: 'grafana_plugin_type',
        magicCookieValue: 'datasource',
      },
      versionedPlugins: {
        2: {
          renderer: new RenderGRPCPluginV2(config, logger),
        },
      },
      logger: logger,
      grpcHost: config.plugin.grpc.host,
      grpcPort: config.plugin.grpc.port,
    });
  } else if (command === 'server') {
    let config: ServiceConfig = defaultServiceConfig;

    if (argv.config) {
      try {
        const fileConfig = readJSONFileSync(argv.config);
        config = _.merge(config, fileConfig);
      } catch (e) {
        console.error('failed to read config from path', argv.config, 'error', e);
        return;
      }
    }

    populateServiceConfigFromEnv(config, env);

    const logger = new ConsoleLogger(config.service.logging);

    const sanitizer = createSanitizer();
    const server = new HttpServer(config, logger, sanitizer);
    await server.start();
  } else {
    console.log('Unknown command');
  }
}

main().catch((err) => {
  const errorLog = {
    '@level': 'error',
    '@message': 'failed to start grafana-image-renderer',
    'error': err.message,
    'trace': err.stack,
  }
  console.error(JSON.stringify(errorLog));
  process.exit(1);
});

function populatePluginConfigFromEnv(config: PluginConfig, env: NodeJS.ProcessEnv) {
  if (env['GF_PLUGIN_GRPC_HOST']) {
    config.plugin.grpc.host = env['GF_PLUGIN_GRPC_HOST'] as string;
  }

  if (env['GF_PLUGIN_GRPC_PORT']) {
    config.plugin.grpc.port = parseInt(env['GF_PLUGIN_GRPC_PORT'] as string, 10);
  }

  if (env['GF_PLUGIN_AUTH_TOKEN']) {
    const authToken = env['GF_PLUGIN_AUTH_TOKEN'] as string;
    config.plugin.security.authToken = authToken.includes(' ') ? authToken.split(' ') : authToken;
  }

  populateRenderingConfigFromEnv(config.rendering, env, true)
}

function populateServiceConfigFromEnv(config: ServiceConfig, env: NodeJS.ProcessEnv) {
  if (env['GF_PLUGIN_RENDERING_CHROME_BIN']) {
    config.rendering.chromeBin = env['GF_PLUGIN_RENDERING_CHROME_BIN'];
  }

  if (env['BROWSER_TZ']) {
    config.rendering.timezone = env['BROWSER_TZ'];
  } else if (env['TZ']) {
    config.rendering.timezone = env['TZ'];
  }

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

  populateRenderingConfigFromEnv(config.rendering, env, false)
}


function populateRenderingConfigFromEnv(config: RenderingConfig, env: NodeJS.ProcessEnv, isPlugin: boolean) {
  const pluginPrefix = isPlugin ? "GF_PLUGIN_" : ""
  const pluginRenderingPrefix = isPlugin ? pluginPrefix + "RENDERING_" : ""

  if (env[pluginPrefix + 'RENDERING_TIMEZONE']) {
    config.timezone = env[pluginPrefix + 'RENDERING_TIMEZONE'];
  } else if (env['BROWSER_TZ']) {
    config.timezone = env['BROWSER_TZ'];
  } else {
    config.timezone = env['TZ'];
  }

  if (env[pluginRenderingPrefix + 'CHROME_BIN']) {
    config.chromeBin = env[pluginRenderingPrefix + 'CHROME_BIN'];
  }

  if (env[pluginPrefix + 'RENDERING_ARGS']) {
    const args = env[pluginPrefix + 'RENDERING_ARGS'] as string;
    if (args.length > 0) {
      const argsList = args.split(',');
      if (argsList.length > 0) {
        config.args = argsList;
      }
    }
  }

  if (env[pluginRenderingPrefix + 'IGNORE_HTTPS_ERRORS']) {
    config.ignoresHttpsErrors = env[pluginRenderingPrefix + 'IGNORE_HTTPS_ERRORS'] === 'true';
  }

  // New for remote
  if (env[pluginPrefix + 'RENDERING_LANGUAGE']) {
    config.acceptLanguage = env[pluginPrefix + 'RENDERING_LANGUAGE'];
  }

  // New for remote
  if (env[pluginPrefix + 'RENDERING_VIEWPORT_WIDTH']) {
    config.width = parseInt(env[pluginPrefix + 'RENDERING_VIEWPORT_WIDTH'] as string, 10);
  }

  // New for remote
  if (env[pluginPrefix + 'RENDERING_VIEWPORT_HEIGHT']) {
    config.height = parseInt(env[pluginPrefix + 'RENDERING_VIEWPORT_HEIGHT'] as string, 10);
  }

  // New for remote
  if (env[pluginPrefix + 'RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR']) {
    config.deviceScaleFactor = parseFloat(env[pluginPrefix + 'RENDERING_VIEWPORT_DEVICE_SCALE_FACTOR'] as string);
  }

  // New for remote
  if (env[pluginPrefix + 'RENDERING_VIEWPORT_MAX_WIDTH']) {
    config.maxWidth = parseInt(env[pluginPrefix + 'RENDERING_VIEWPORT_MAX_WIDTH'] as string, 10);
  }

  // New for remote
  if (env[pluginPrefix + 'RENDERING_VIEWPORT_MAX_HEIGHT']) {
    config.maxHeight = parseInt(env[pluginPrefix + 'RENDERING_VIEWPORT_MAX_HEIGHT'] as string, 10);
  }

  if (env[pluginPrefix + 'RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR']) {
    config.maxDeviceScaleFactor = parseFloat(env[pluginPrefix + 'RENDERING_VIEWPORT_MAX_DEVICE_SCALE_FACTOR'] as string);
  }

  if (env[pluginPrefix + 'RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL']) {
    config.pageZoomLevel = parseFloat(env[pluginPrefix + 'RENDERING_VIEWPORT_PAGE_ZOOM_LEVEL'] as string);
  }

  if (env[pluginPrefix + 'RENDERING_MODE']) {
    config.mode = env[pluginPrefix + 'RENDERING_MODE'] as string;
  }

  if (env[pluginPrefix + 'RENDERING_CLUSTERING_MODE']) {
    config.clustering.mode = env[pluginPrefix + 'RENDERING_CLUSTERING_MODE'] as string;
  }

  if (env[pluginPrefix + 'RENDERING_CLUSTERING_MAX_CONCURRENCY']) {
    config.clustering.maxConcurrency = parseInt(env[pluginPrefix + 'RENDERING_CLUSTERING_MAX_CONCURRENCY'] as string, 10);
  }

  if (env[pluginPrefix + 'RENDERING_CLUSTERING_TIMEOUT']) {
    config.clustering.timeout = parseInt(env[pluginPrefix + 'RENDERING_CLUSTERING_TIMEOUT'] as string, 10);
  }

  if (env[pluginPrefix + 'RENDERING_VERBOSE_LOGGING']) {
    config.verboseLogging = env[pluginPrefix + 'RENDERING_VERBOSE_LOGGING'] === 'true';
  }

  if (env[pluginPrefix + 'RENDERING_DUMPIO']) {
    config.dumpio = env[pluginPrefix + 'RENDERING_DUMPIO'] === 'true';
  }
}
