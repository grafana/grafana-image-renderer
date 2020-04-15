import * as os from 'os';
import * as uniqueFilename from 'unique-filename';
import * as puppeteer from 'puppeteer';
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

export class Browser {
  constructor(protected config: RenderingConfig, protected log: Logger) {
    this.log.info(
      'Browser initiated',
      'chromeBin',
      this.config.chromeBin,
      'ignoresHttpsErrors',
      this.config.ignoresHttpsErrors,
      'timezone',
      this.config.timezone,
      'args',
      this.config.args,
      'dumpio',
      this.config.dumpio,
      'verboseLogging',
      this.config.verboseLogging
    );
  }

  async getBrowserVersion(): Promise<string> {
    let browser;

    try {
      const launcherOptions = this.getLauncherOptions({});
      browser = await puppeteer.launch(launcherOptions);
      return await browser.version();
    } finally {
      if (browser) {
        await browser.close();
      }
    }
  }

  async start(): Promise<void> {}

  validateOptions(options: RenderOptions) {
    if (options.url.startsWith(`socket://`)) {
      // Puppeteer doesn't support socket:// URLs
      throw new Error(`Image rendering in socket mode is not supported`);
    }
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
      dumpio: this.config.dumpio,
      args: this.config.args,
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
      browser = await puppeteer.launch(launcherOptions);
      page = await browser.newPage();
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
    await page.mouse.move(options.width, options.height);
    this.log.debug(`Navigating to ${options.url}`);
    await page.goto(options.url, { waitUntil: 'networkidle0' });
    await page.waitForFunction(
      () => {
        const panelCount = document.querySelectorAll('.panel').length || document.querySelectorAll('.panel-container').length;
        return (window as any).panelsRendered >= panelCount;
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
    this.log.debug('Browser request', 'url', req.url(), 'method', req.method());
  };

  logRequestFailed = (req: any) => {
    this.log.error('Browser request failed', 'url', req.url(), 'method', req.method(), 'failure', req.failure().errorText);
  };

  logRequestFinished = (req: any) => {
    this.log.debug('Browser request finished', 'url', req.url(), 'method', req.method());
  };

  logPageClosed = () => {
    this.log.debug('Browser page closed');
  };
}
