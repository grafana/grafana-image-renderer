import * as os from 'os';
import * as uniqueFilename from 'unique-filename';
import * as puppeteer from 'puppeteer';
import * as chokidar from 'chokidar';
import * as path from 'path';
import * as fs from 'fs';
import { Logger } from '../logger';
import { RenderingConfig } from '../config';

export interface HTTPHeaders {
  'Accept-Language'?: string;
  [header: string]: string | undefined;
}

export enum RenderType {
  CSV = 'csv',
  PNG = 'png',
}

export interface RenderOptions {
  renderType: RenderType;
  url: string;
  width: string | number;
  height: string | number;
  filePath: string;
  timeout: string | number;
  renderKey: string;
  domain: string;
  timezone?: string;
  encoding?: string;
  deviceScaleFactor?: string | number;
  headers?: HTTPHeaders;
}

export interface RenderResponse {
  filePath: string;
  fileName?: string;
}

export class Browser {
  constructor(protected config: RenderingConfig, protected log: Logger) {
    this.log.debug('Browser initialized', 'config', this.config);
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

    options.headers = options.headers || {};
    const headers = {};

    if (options.headers['Accept-Language']) {
      headers['Accept-Language'] = options.headers['Accept-Language'];
    } else if (this.config.acceptLanguage) {
      headers['Accept-Language'] = this.config.acceptLanguage;
    }

    options.headers = headers;

    if (options.renderType === RenderType.CSV) {
      return;
    }

    options.width = parseInt(options.width as string, 10) || this.config.width;
    options.height = parseInt(options.height as string, 10) || this.config.height;
    options.timeout = parseInt(options.timeout as string, 10) || 30;

    if (options.width < 10) {
      options.width = this.config.width;
    }

    if (options.width > this.config.maxWidth) {
      options.width = this.config.maxWidth;
    }

    if (options.height < 10) {
      options.height = this.config.height;
    }

    if (options.height > this.config.maxHeight) {
      options.height = this.config.maxHeight;
    }

    options.deviceScaleFactor = parseFloat(((options.deviceScaleFactor as string) || '1') as string) || 1;

    if (options.deviceScaleFactor > this.config.maxDeviceScaleFactor) {
      options.deviceScaleFactor = this.config.deviceScaleFactor;
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

      switch (options.renderType) {
        case RenderType.CSV:
          return await this.exportCSV(page, options);
        case RenderType.PNG:
        default:
          return await this.takeScreenshot(page, options);
      }
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

  async exportCSV(page: any, options: any): Promise<RenderResponse> {
    if (this.config.verboseLogging) {
      this.log.debug('Setting cookie for page', 'renderKey', options.renderKey, 'domain', options.domain);
    }
    await page.setCookie({
      name: 'renderKey',
      value: options.renderKey,
      domain: options.domain,
    });

    if (options.headers && Object.keys(options.headers).length > 0) {
      this.log.debug(`Setting extra HTTP headers for page`, 'headers', options.headers);
      await page.setExtraHTTPHeaders(options.headers);
    }

    const downloadPath = uniqueFilename(os.tmpdir());
    fs.mkdirSync(downloadPath);
    const watcher = chokidar.watch(downloadPath);
    let downloadFilePath = '';
    watcher.on('add', file => {
      if (!file.endsWith('.crdownload')) {
        downloadFilePath = file;
      }
    });

    await page._client.send('Page.setDownloadBehavior', { behavior: 'allow', downloadPath: downloadPath });

    if (this.config.verboseLogging) {
      this.log.debug('Navigating and waiting for all network requests to finish', 'url', options.url);
    }

    await page.goto(options.url, { waitUntil: 'networkidle0', timeout: options.timeout * 1000 });

    if (this.config.verboseLogging) {
      this.log.debug('Waiting for download to end');
    }

    const startDate = Date.now();
    while (Date.now() - startDate <= options.timeout * 1000) {
      if (downloadFilePath != '') {
        break;
      }
      await new Promise(resolve => setTimeout(resolve, 500));
    }

    await watcher.close();

    let filePath = downloadFilePath;
    if (options.filePath) {
      fs.renameSync(downloadFilePath, options.filePath);
      filePath = options.filePath;
    }

    return { filePath, fileName: path.basename(downloadFilePath) };
  }

  async takeScreenshot(page: any, options: any): Promise<RenderResponse> {
    if (this.config.verboseLogging) {
      this.log.debug(
        'Setting viewport for page',
        'width',
        options.width.toString(),
        'height',
        options.height.toString(),
        'deviceScaleFactor',
        options.deviceScaleFactor.toString()
      );
    }
    await page.setViewport({
      width: options.width,
      height: options.height,
      deviceScaleFactor: options.deviceScaleFactor,
    });

    if (this.config.verboseLogging) {
      this.log.debug('Setting cookie for page', 'renderKey', options.renderKey, 'domain', options.domain);
    }
    await page.setCookie({
      name: 'renderKey',
      value: options.renderKey,
      domain: options.domain,
    });

    if (options.headers && Object.keys(options.headers).length > 0) {
      this.log.debug(`Setting extra HTTP headers for page`, 'headers', options.headers);
      await page.setExtraHTTPHeaders(options.headers);
    }

    if (this.config.verboseLogging) {
      this.log.debug('Moving mouse on page', 'x', options.width, 'y', options.height);
    }
    await page.mouse.move(options.width, options.height);

    if (this.config.verboseLogging) {
      this.log.debug('Navigating and waiting for all network requests to finish', 'url', options.url);
    }

    await page.goto(options.url, { waitUntil: 'networkidle0', timeout: options.timeout * 1000 });

    if (this.config.verboseLogging) {
      this.log.debug('Waiting for dashboard/panel to load', 'timeout', `${options.timeout}s`);
    }
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

    if (this.config.verboseLogging) {
      this.log.debug('Taking screenshot', 'filePath', options.filePath);
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
    this.log.error('Browser uncaught exception', 'error', err.stack);
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
