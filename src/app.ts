import { GrpcPlugin } from './grpc-plugin';
import { HttpServer } from './http-server';
import { Logger, ConsoleLogger, PluginLogger } from './logger';
import { Browser } from './browser';
import * as minimist from 'minimist';


async function main() {
  const argv = minimist(process.argv.slice(2));
  const command = argv._[0];

  if (command === undefined) {
    const logger = new PluginLogger();
    const browser = new Browser(logger);
    const plugin = new GrpcPlugin(logger, browser);
    plugin.start();
  } else if (command === 'server') {
    if (!argv.port) {
      console.log('Specify http port using --port=5000');
      return;
    }

    const logger = new ConsoleLogger();
    const browser = new Browser(logger);
    const server = new HttpServer({port: argv.port}, logger, browser);

    server.start();

  } else {
    console.log('Unknown command');
  }
}

main();


// const puppeteer = require('puppeteer');
//
// var argv = require('minimist')(process.argv.slice(2));
// console.dir(argv);
//
// (async () => {
//   const browser = await puppeteer.launch();
//   const page = await browser.newPage();
//   await page.goto('http://localhost:3000');
//   await page.screenshot({path: 'example.png'});
//
//   await browser.close();
// })();
