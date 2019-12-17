import * as path from 'path';
import * as puppeteer from 'puppeteer';
import * as _ from 'lodash';
import { GrpcPlugin } from './plugin/grpc-plugin';
import { HttpServer } from './service/http-server';
import { ConsoleLogger, PluginLogger } from './logger';
import { Browser } from './browser';
import * as minimist from 'minimist';
import { defaultPluginConfig, defaultServiceConfig, readJSONFileSync } from './config';

async function main() {
  const argv = minimist(process.argv.slice(2));
  const env = Object.assign({}, process.env);
  const command = argv._[0];

  if (command === undefined) {
    const config = defaultPluginConfig;

    if (env['GF_RENDERER_PLUGIN_IGNORE_HTTPS_ERRORS']) {
      config.rendering.ignoresHttpsErrors = env['GF_RENDERER_PLUGIN_IGNORE_HTTPS_ERRORS'] === 'true';
    }

    if (env['GF_RENDERER_PLUGIN_CHROME_BIN']) {
      config.rendering.chromeBin = env['GF_RENDERER_PLUGIN_CHROME_BIN'];
    } else if ((process as any).pkg) {
      const parts = puppeteer.executablePath().split(path.sep);
      while (!parts[0].startsWith('chrome-')) {
        parts.shift();
      }

      config.rendering.chromeBin = [path.dirname(process.execPath), ...parts].join(path.sep);
    }

    const logger = new PluginLogger();
    const browser = new Browser(config.rendering, logger);
    const plugin = new GrpcPlugin(config, logger, browser);
    plugin.start();
  } else if (command === 'server') {
    let config = defaultServiceConfig;
    const logger = new ConsoleLogger();

    if (argv.config) {
      try {
        const fileConfig = readJSONFileSync(argv.config);
        config = _.merge(config, fileConfig);
      } catch (e) {
        logger.error('failed to read config from path', argv.config, 'error', e);
        return;
      }
    }

    if (env['IGNORE_HTTPS_ERRORS']) {
      config.rendering.ignoresHttpsErrors = env['IGNORE_HTTPS_ERRORS'] === 'true';
    }

    if (env['CHROME_BIN']) {
      config.rendering.chromeBin = env['CHROME_BIN'];
    }

    if (env['ENABLE_METRICS']) {
      config.service.metrics.enabled = env['ENABLE_METRICS'] === 'true';
    }

    const browser = new Browser(config.rendering, logger);
    const server = new HttpServer(config, logger, browser);

    server.start();
  } else {
    console.log('Unknown command');
  }
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
