import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Logger } from './logger';
import uniqueFilename = require('unique-filename');

export class Browser {

  constructor(private log: Logger) {
  }

  validateOptions(options) {
    options.width = parseInt(options.width) || 1000;
    options.height = parseInt(options.height) || 500;
    options.timeout = parseInt(options.timeout) || 30;

    if (options.width > 3000 || options.width < 10) {
      options.width = 2500;
    }

    if (options.height > 3000 || options.height < 10) {
      options.height = 1500;
    }
  }

  async render(options) {
    let browser;
    let page;
    let exploreHandle;
    const env = Object.assign({}, process.env);

    try {
      this.validateOptions(options);

      // set env timezone
      env.TZ = options.timezone || process.env.TZ;

      browser = await puppeteer.launch({env: env, args: ['--no-sandbox']});
      page = await browser.newPage();

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

      // wait for all panels to render
      await page.waitForFunction(() => {
        const panelCount = document.querySelectorAll('.panel').length;
        const exploreCount = document.querySelectorAll('.explore-graph > canvas').length;
        return exploreCount > 0 || ((<any>window).panelsRendered >= panelCount);
      }, {
        timeout: options.timeout * 1000
      });

      if (!options.filePath) {
        options.filePath = uniqueFilename(os.tmpdir()) + '.png';
      }

      exploreHandle = await page.$('.explore-container .panel-container:first-of-type');
      if (exploreHandle != null) {
        await exploreHandle.screenshot({path: options.filePath});
      } else {
        await page.screenshot({path: options.filePath});
      }

      return { filePath: options.filePath };

    } finally {
      if (page) {
        await page.close();
      }
      if (browser) {
        await browser.close();
      }
    }
  }
}

