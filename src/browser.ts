import * as os from 'os';
import * as puppeteer from 'puppeteer';
import * as boom from 'boom';
import { Logger } from './logger';
import uniqueFilename = require('unique-filename');
import { RenderingConfig } from './config';

const allowedFormats: string[] = ['png', 'jpeg', 'pdf'];

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

    if (options.encoding === '') {
      options.encoding = 'png';
    }

    if (allowedFormats.indexOf(options.encoding) === -1) {
      throw boom.badRequest('Unsupported encoding ' + options.encoding);
    }

    if (options.jsonData) {
      options.jsonData = JSON.parse(options.jsonData);
    } else {
      options.jsonData = {};
    }
  }

  async render(options) {
    let browser;
    let page;
    const env = Object.assign({}, process.env);

    try {
      this.validateOptions(options);

      this.log.debug('JSON data: %j', options.jsonData);

      // set env timezone
      env.TZ = options.timezone || process.env.TZ;

      const launcherOptions: any = {
        env: env,
        ignoreHTTPSErrors: this.config.ignoresHttpsErrors,
        args: ['--no-sandbox'],
        ...options.jsonData.launchOptions,
      };

      if (this.config.chromeBin) {
        launcherOptions.executablePath = this.config.chromeBin;
      }

      browser = await puppeteer.launch(launcherOptions);
      page = await browser.newPage();

      await page.setViewport({
        width: options.width,
        height: options.height,
        ...options.jsonData.viewport,
      });

      if (options.jsonData.emulateMedia) {
        await page.emulateMedia(options.jsonData.emulateMedia);
      }

      if (options.jsonData.defaultNavigationTimeout) {
        await page.setDefaultNavigationTimeout(options.jsonData.defaultNavigationTimeout);
      }

      await page.setCookie({
        name: 'renderKey',
        value: options.renderKey,
        domain: options.domain,
      });

      // build url
      let url = options.url + (options.jsonData.extraUrlParams ? options.jsonData.extraUrlParams : '');

      // wait until all data was loaded
      this.log.debug('Goto url: %j', url);
      await page.goto(url, { waitUntil: 'networkidle0' });

      // extra javascript
      if (options.jsonData.scriptTags instanceof Array) {
        for (let val of options.jsonData.scriptTags) {
          await page.addScriptTag(val);
        }
      }

      // extra style tags
      if (options.jsonData.styleTags instanceof Array) {
        for (let val of options.jsonData.styleTags) {
          await page.addStyleTag(val);
        }
      }

      // wait for all panels to render
      await page.waitForFunction(
        () => {
          const panelCount = document.querySelectorAll('.panel').length || document.querySelectorAll('.panel-container').length;
          return (window as any).panelsRendered >= panelCount;
        },
        {
          timeout: options.timeout * 1000,
        }
      );

      // extra wait
      if (options.jsonData.waitFor) {
        await page.waitFor(options.jsonData.waitFor);
      }

      if (!options.filePath) {
        options.filePath = uniqueFilename(os.tmpdir()) + '.' + options.encoding;
      }

      if (options.encoding === 'pdf') {
        await page.pdf({ path: options.filePath, ...options.jsonData.pdf });
      } else {
        await page.screenshot({ path: options.filePath, type: options.encoding, ...options.jsonData.screenshot });
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
