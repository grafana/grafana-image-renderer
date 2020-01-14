import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Logger } from './logger';
import uniqueFilename = require('unique-filename');
import { RenderingConfig } from './config';

export class Browser {
  chromeBin?: string;
  ignoreHTTPSErrors: boolean;

  constructor(private config: RenderingConfig, private log: Logger) {}

  validateOptions(options) {
    options.width = parseInt(options.width, 10) || 1000;
    options.height = parseInt(options.height, 10) || 500;
    options.timeout = parseInt(options.timeout, 10) || 30;

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

      const launcherOptions: any = {
        env: env,
        ignoreHTTPSErrors: this.config.ignoresHttpsErrors,
        args: ['--no-sandbox'],
      };

      if (this.config.chromeBin) {
        launcherOptions.executablePath = this.config.chromeBin;
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

      // wait until all data was loaded
      await page.goto(options.url, { waitUntil: 'networkidle0' });

      // wait for all panels (or any alert panel) to render
      await page.waitForFunction(
        () => {
          const panelCount = document.querySelectorAll('.panel').length || document.querySelectorAll('.panel-container').length;
          return ((window as any).panelsRendered >= panelCount) || (document.querySelectorAll('.alert').length > 0);
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
