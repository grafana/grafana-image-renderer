import * as path from 'path';
import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Logger } from './logger';
import uniqueFilename = require('unique-filename');

export function newPluginBrowser(log: Logger): Browser {
  const env = Object.assign({}, process.env);
  let chromeBin: any;

  if (env['GF_RENDERER_PLUGIN_CHROME_BIN']) {
    chromeBin = env['GF_RENDERER_PLUGIN_CHROME_BIN'];
  } else if ((process as any).pkg) {
    const parts = puppeteer.executablePath().split(path.sep);
    while (!parts[0].startsWith('chrome-')) {
      parts.shift();
    }

    chromeBin = [path.dirname(process.execPath), ...parts].join(path.sep);
  }

  return new Browser(log, chromeBin);
}

export function newServerBrowser(log: Logger): Browser {
  const env = Object.assign({}, process.env);
  let chromeBin: any;

  if (env['CHROME_BIN']) {
    chromeBin = env['CHROME_BIN'];
  }

  return new Browser(log, chromeBin);
}

export class Browser {
  chromeBin?: string;

  constructor(private log: Logger, chromeBin?: string) {
    this.chromeBin = chromeBin;
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
    const env = Object.assign({}, process.env);

    try {
      this.validateOptions(options);

      // set env timezone
      env.TZ = options.timezone || process.env.TZ;

      let launcherOptions: any = {
        env: env,
        args: ['--no-sandbox'],
      };

      if (this.chromeBin) {
        launcherOptions.executablePath = this.chromeBin;
      }

      browser = await puppeteer.launch(launcherOptions);
      page = await browser.newPage();

      await page.setViewport({
        width: options.width,
        height: options.height,
        deviceScaleFactor: 1,
      });

      await page.setCookie({
        name: 'renderKey',
        value: options.renderKey,
        domain: options.domain,
      });

      await page.goto(options.url);

      // wait for all panels to render
      await page.waitForFunction(
        () => {
          const panelCount =
            document.querySelectorAll('.panel').length || document.querySelectorAll('.panel-container').length;
          return (<any>window).panelsRendered >= panelCount;
        },
        {
          timeout: options.timeout * 1000,
        }
      );

      if (!options.filePath) {
        options.filePath = uniqueFilename(os.tmpdir()) + '.png';
      }

      await page.screenshot({ path: options.filePath });

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
