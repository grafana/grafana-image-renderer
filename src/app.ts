import { startTracing } from './tracing';
import path from 'path';
import _ from 'lodash';
import fs from 'fs';
import { Browser, computeExecutablePath } from '@puppeteer/browsers';
import { RenderGRPCPluginV2 } from './plugin/v2/grpc_plugin';
import { HttpServer } from './service/http-server';
import { ServiceConfig } from './service/config';
import { PluginConfig } from './plugin/v2/config';
import { ConsoleLogger, PluginLogger } from './logger';
import minimist from 'minimist';
import { serve } from './node-plugin';
import { createSanitizer } from './sanitizer/Sanitizer';
import { getConfig } from './config/config';

async function main() {
  const argv = minimist(process.argv.slice(2));
  const command = argv._[0];

  if (command === undefined) {
    const config = getConfig() as PluginConfig;
    const logger = new PluginLogger();

    if (config.rendering.tracing.url) {
      startTracing(logger);
    }

    if (!config.rendering.chromeBin && (process as any).pkg) {
      const execPath = path.dirname(process.execPath);
      const chromeInfoFile = fs.readFileSync(path.resolve(execPath, 'chrome-info.json'), 'utf8');
      const chromeInfo = JSON.parse(chromeInfoFile);

      config.rendering.chromeBin = computeExecutablePath({
        cacheDir: path.dirname(process.execPath),
        browser: Browser.CHROMEHEADLESSSHELL,
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
    const config = getConfig() as ServiceConfig;
    const logger = new ConsoleLogger(config.service.logging);

    if (config.rendering.tracing.url) {
      startTracing(logger);
    }

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
    error: err.message,
    trace: err.stack,
  };
  console.error(JSON.stringify(errorLog));
  process.exit(1);
});
