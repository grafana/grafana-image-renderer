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
      deviceScaleFactor: 1,
    });

    await page.setCookie({
      'name': 'renderKey',
      'value': options.renderKey,
      'domain': options.domain,
    });

    await page.goto(options.url);

    if (!options.filePath) {
      options.filePath = uniqueFilename(os.tmpdir()) + '.png';
    }

    await page.screenshot({path: options.filePath});
    page.close();

    return { filePath: options.filePath };
  }
}

