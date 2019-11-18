import { GrpcPlugin } from './grpc-plugin';
import { HttpServer } from './http-server';
import { ConsoleLogger, PluginLogger } from './logger';
import { newPluginBrowser, newServerBrowser } from './browser';
import * as minimist from 'minimist';

async function main() {
  const argv = minimist(process.argv.slice(2));
  const command = argv._[0];

  if (command === undefined) {
    const logger = new PluginLogger();
    const browser = newPluginBrowser(logger);
    const plugin = new GrpcPlugin(logger, browser);
    plugin.start();
  } else if (command === 'server') {
    const logger = new ConsoleLogger();

    if (!argv.port) {
      logger.error('Specify http port using argument --port=5000');
      return;
    }

    const browser = newServerBrowser(logger);
    const server = new HttpServer({ port: argv.port }, logger, browser);

    server.start();
  } else {
    console.log('Unknown command');
  }
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
