import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Logger } from './logger';
import { registerExitCleanUp } from './exit';
import uniqueFilename = require('unique-filename');

export class Browser {
  private instance;

  constructor(private log: Logger) {
  }

  async start() {
    this.log.info("Starting chrome");
    this.instance = await puppeteer.launch();

    registerExitCleanUp(() => {
      this.instance.close();
    });
  }

  validateOptions(options) {
    options.width = parseInt(options.width) || 1000;
    options.height = parseInt(options.height) || 500;

    if (options.width > 3000 || options.width < 10) {
      options.width = 2500;
    }

    if (options.height > 3000 || options.height < 10) {
      options.height = 1500;
    }
  }

  async render(options) {
    const page = await this.instance.newPage();

    this.validateOptions(options);

    await page.setViewport({
      width: options.width,
      height: options.height,
    });

    if (!options.url) {
      options.url = "http://localhost:3000/d-solo/000000088/testdata-graph-panel-last-1h?orgId=1&panelId=4&from=1526537735449&to=1526541335449&width=1000&height=500&tz=UTC%2B02:00";
    }

    await page.goto(options.url);

    if (!options.filePath) {
      options.filePath = uniqueFilename(os.tmpdir()) + '.png';
    }

    await page.screenshot({path: options.filePath});
    page.close();

    return { filePath: options.filePath };
  }
}

