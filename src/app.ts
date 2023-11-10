import * as path from 'path';
import * as _ from 'lodash';
import * as fs from 'fs';
import { Browser, computeExecutablePath } from '@puppeteer/browsers';
import { RenderGRPCPluginV2 } from './plugin/v2/grpc_plugin';
import { HttpServer } from './service/http-server';
import { populateServiceConfigFromEnv, ServiceConfig, defaultServiceConfig } from './service/config';
import { populatePluginConfigFromEnv, PluginConfig, defaultPluginConfig } from './plugin/v2/config';
import { ConsoleLogger, PluginLogger } from './logger';
import * as minimist from 'minimist';
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

function readJSONFileSync(filePath: string) {
  const rawdata = fs.readFileSync(filePath, 'utf8');
  return JSON.parse(rawdata);
};

