import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Browser as PuppeteerBrowser, Page } from 'puppeteer';
import uniqueFilename = require('unique-filename');
import { Logger } from '../logger';
import { RenderingConfig } from '../config';

export interface RenderResponse {
  filePath: string;
}

export interface BrowserTimings {
  launch(callback: () => Promise<PuppeteerBrowser>): Promise<PuppeteerBrowser>;
  newPage(callback: () => Promise<Page>): Promise<Page>;
  navigate(callback: () => Promise<void>): Promise<void>;
  panelsRendered(callback: () => Promise<void>): Promise<void>;
  screenshot(callback: () => Promise<void>): Promise<void>;
}

export class NoOpBrowserTiming {
  async launch(callback: () => Promise<PuppeteerBrowser>) {
    return await callback();
  }

  async newPage(callback: () => Promise<void>) {
    return await callback();
  }

  async navigate(callback: () => Promise<void>) {
    return await callback();
  }

  async panelsRendered(callback: () => Promise<void>) {
    return await callback();
  }

  async screenshot(callback: () => Promise<void>) {
    return await callback();
  }
}

export class Browser {
  constructor(protected config: RenderingConfig, protected log: Logger, protected timings: BrowserTimings) {}

  async getBrowserVersion(): Promise<string> {
    const launcherOptions = this.getLauncherOptions({});
    const browser = await puppeteer.launch(launcherOptions);
    return browser.version();
  }

  async start(): Promise<void> {}

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

  getLauncherOptions(options) {
    const env = Object.assign({}, process.env);
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

    return launcherOptions;
  }

  async render(options): Promise<RenderResponse> {
    let browser;
    let page: any;

    try {
      this.validateOptions(options);
      const launcherOptions = this.getLauncherOptions(options);

      browser = await this.timings.launch(
        async () =>
          // launch browser
          await puppeteer.launch(launcherOptions)
      );
      page = await this.timings.newPage(
        async () =>
          // open a new page
          await browser.newPage()
      );

      return await this.takeScreenshot(page, options);
    } finally {
      if (page) {
        await page.close();
      }
      if (browser) {
        await browser.close();
      }
    }
  }

  async takeScreenshot(page: any, options: any): Promise<RenderResponse> {
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

    await this.timings.navigate(async () => {
      // wait until all data was loaded
      await page.goto(options.url, { waitUntil: 'networkidle0' });
    });

    await this.timings.panelsRendered(async () => {
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
    });

    if (!options.filePath) {
      options.filePath = uniqueFilename(os.tmpdir()) + '.png';
    }

    await this.timings.screenshot(async () => {
      await page.screenshot({ path: options.filePath });
    });

    return { filePath: options.filePath };
  }
}
