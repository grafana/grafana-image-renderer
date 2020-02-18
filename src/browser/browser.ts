import * as os from 'os';
import * as puppeteer from 'puppeteer';
import { Browser as PuppeteerBrowser, Page } from 'puppeteer';
import uniqueFilename = require('unique-filename');
import { Logger } from '../logger';
import { RenderingConfig } from '../config';

export interface RenderOptions {
  url: string;
  width: string | number;
  height: string | number;
  filePath: string;
  timeout: string | number;
  renderKey: string;
  domain: string;
  timezone?: string;
  encoding?: string;
}

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

  validateOptions(options: RenderOptions) {
    options.width = parseInt(options.width as string, 10) || 1000;
    options.height = parseInt(options.height as string, 10) || 500;
    options.timeout = parseInt(options.timeout as string, 10) || 30;

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
    env.TZ = options.timezone || this.config.timezone;

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

  async render(options: RenderOptions): Promise<RenderResponse> {
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

      this.addPageListeners(page);

      return await this.takeScreenshot(page, options);
    } finally {
      if (page) {
        this.removePageListeners(page);
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

  addPageListeners(page: any) {
    page.on('error', this.logError);
    page.on('pageerror', this.logPageError);
    page.on('requestfailed', this.logRequestFailed);
    page.on('console', this.logConsoleMessage);

    if (this.config.verboseLogging) {
      page.on('request', this.logRequest);
      page.on('requestfinished', this.logRequestFinished);
      page.on('close', this.logPageClosed);
    }
  }

  removePageListeners(page: any) {
    page.removeListener('error', this.logError);
    page.removeListener('pageerror', this.logPageError);
    page.removeListener('requestfailed', this.logRequestFailed);
    page.removeListener('console', this.logConsoleMessage);

    if (this.config.verboseLogging) {
      page.removeListener('request', this.logRequest);
      page.removeListener('requestfinished', this.logRequestFinished);
      page.removeListener('close', this.logPageClosed);
    }
  }

  logError = (err: Error) => {
    this.log.error('Browser page crashed', 'error', err.toString());
  };

  logPageError = (err: Error) => {
    this.log.error('Browser uncaught exception', 'error', err.toString());
  };

  logConsoleMessage = (msg: any) => {
    const msgType = msg.type();
    if (!this.config.verboseLogging && msgType !== 'error') {
      return;
    }

    const loc = msg.location();
    if (msgType === 'error') {
      this.log.error('Browser console error', 'msg', msg.text(), 'url', loc.url, 'line', loc.lineNumber, 'column', loc.columnNumber);
      return;
    }

    this.log.debug(`Browser console ${msgType}`, 'msg', msg.text(), 'url', loc.url, 'line', loc.lineNumber, 'column', loc.columnNumber);
  };

  logRequest = (req: any) => {
    this.log.debug('Browser request', 'url', req._url, 'method', req._url);
  };

  logRequestFailed = (req: any) => {
    this.log.error('Browser request failed', 'url', req._url, 'method', req._method);
  };

  logRequestFinished = (req: any) => {
    this.log.debug('Browser request finished', 'url', req._url, 'method', req._method);
  };

  logPageClosed = () => {
    this.log.debug('Browser page closed');
  };
}
