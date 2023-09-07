import * as os from 'os';
import * as uniqueFilename from 'unique-filename';
import * as puppeteer from 'puppeteer';
import * as chokidar from 'chokidar';
import * as path from 'path';
import * as fs from 'fs';
import * as promClient from 'prom-client';
import * as Jimp from 'jimp';
import { Logger } from '../logger';
import { RenderingConfig } from '../config';
import { ImageRenderOptions, RenderOptions } from '../types';

export interface Metrics {
  durationHistogram: promClient.Histogram;
}
export interface RenderResponse {
  filePath: string;
}

export interface RenderCSVResponse {
  filePath: string;
  fileName?: string;
}

type DashboardScrollingResult = { scrolled: false } | { scrolled: true; scrollHeight: number };

type PuppeteerLaunchOptions = Parameters<typeof puppeteer['launch']>[0];

export class Browser {
  constructor(protected config: RenderingConfig, protected log: Logger, protected metrics: Metrics) {
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
        const pages = await browser.pages();
        await Promise.all(pages.map((page) => page.close()));
        await browser.close();
      }
    }
  }

  async start(): Promise<void> {}

  validateRenderOptions(options: RenderOptions) {
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

    if (typeof options.timeout === 'string') {
      options.timeout = parseInt(options.timeout as unknown as string, 10);
    }

    options.timeout = options.timeout || 30;
  }

  validateImageOptions(options: ImageRenderOptions) {
    this.validateRenderOptions(options);

    options.width = parseInt(options.width as string, 10) || this.config.width;
    options.height = parseInt(options.height as string, 10) || this.config.height;

    if (options.width < 10) {
      options.width = this.config.width;
    }

    if (options.width > this.config.maxWidth) {
      options.width = this.config.maxWidth;
    }

    // Trigger full height snapshots with a negative height value
    if (options.height === -1) {
      options.fullPageImage = true;
      options.height = Math.floor(options.width * 0.75);
    }

    if (options.height < 10) {
      options.height = this.config.height;
    }

    if (options.height > this.config.maxHeight) {
      options.height = this.config.maxHeight;
    }

    options.deviceScaleFactor = parseFloat(((options.deviceScaleFactor as string) || '1') as string) || 1;

    // Scaled thumbnails
    if (options.deviceScaleFactor <= 0) {
      options.scaleImage = options.deviceScaleFactor * -1;
      options.deviceScaleFactor = 1;

      if (options.scaleImage > 1) {
        options.width *= options.scaleImage;
        options.height *= options.scaleImage;
      } else {
        options.scaleImage = undefined;
      }
    } else if (options.deviceScaleFactor > this.config.maxDeviceScaleFactor) {
      options.deviceScaleFactor = this.config.deviceScaleFactor;
    }
  }

  getLauncherOptions(options) {
    const env = Object.assign({}, process.env);
    // set env timezone
    env.TZ = options.timezone || this.config.timezone;

    const launcherOptions: PuppeteerLaunchOptions = {
      env: env,
      ignoreHTTPSErrors: this.config.ignoresHttpsErrors,
      dumpio: this.config.dumpio,
      args: this.config.args,
    };

    if (this.config.chromeBin) {
      launcherOptions.executablePath = this.config.chromeBin;
    }

    launcherOptions.headless = !this.config.headed ? "new" : false;

    return launcherOptions;
  }

  async setTimezone(page: puppeteer.Page, options: RenderOptions) {
    const timezone = options.timezone || this.config.timezone;
    if (timezone) {
      await page.emulateTimezone(timezone);
    }
  }

  async preparePage(page: puppeteer.Page, options: RenderOptions) {
    if (this.config.emulateNetworkConditions && this.config.networkConditions) {
      const client = await page.target().createCDPSession();
      await client.send('Network.emulateNetworkConditions', this.config.networkConditions);
    }

    if (options.renderKey) {
      if (this.config.verboseLogging) {
        this.log.debug('Setting cookie for page', 'renderKey', options.renderKey, 'domain', options.domain);
      }
      await page.setCookie({
        name: 'renderKey',
        value: options.renderKey,
        domain: options.domain,
      });
    }

    if (options.headers && Object.keys(options.headers).length > 0) {
      this.log.debug(`Setting extra HTTP headers for page`, 'headers', options.headers);
      await page.setExtraHTTPHeaders(options.headers as any);
    }

    // automatically accept "Changes you made may not be saved" dialog which could be triggered by saving migrated dashboard schema
    const acceptBeforeUnload = (dialog) => dialog.type() === 'beforeunload' && dialog.accept();
    page.on('dialog', acceptBeforeUnload);
  }

  async scrollToLoadAllPanels(page: puppeteer.Page, options: ImageRenderOptions): Promise<DashboardScrollingResult> {
    const scrollDivSelector = '[class="scrollbar-view"]';
    const scrollDelay = options.scrollDelay ?? 500;

    await page.waitForSelector(scrollDivSelector);
    const heights: { dashboard?: { scroll: number; client: number }; body: { client: number } } = await page.evaluate((scrollDivSelector) => {
      const body = { client: document.body.clientHeight };
      const dashboardDiv = document.querySelector(scrollDivSelector);
      if (!dashboardDiv) {
        return {
          body,
        };
      }

      return {
        dashboard: { scroll: dashboardDiv.scrollHeight, client: dashboardDiv.clientHeight },
        body,
      };
    }, scrollDivSelector);

    if (!heights.dashboard) {
      return {
        scrolled: false,
      };
    }

    if (heights.dashboard.scroll <= heights.dashboard.client) {
      return {
        scrolled: false,
      };
    }

    const scrolls = Math.floor(heights.dashboard.scroll / heights.dashboard.client);

    for (let i = 0; i < scrolls; i++) {
      await page.evaluate(
        (scrollByHeight, scrollDivSelector) => {
          document.querySelector(scrollDivSelector)?.scrollBy(0, scrollByHeight);
        },
        heights.dashboard.client,
        scrollDivSelector
      );
      
      await new Promise(executor => setTimeout(executor, scrollDelay));
    }

    await page.evaluate((scrollDivSelector) => {
      document.querySelector(scrollDivSelector)?.scrollTo(0, 0);
    }, scrollDivSelector);

    // Header height will be equal to 0 in Kiosk mode
    const headerHeight = heights.body.client - heights.dashboard.client;
    return {
      scrolled: true,
      scrollHeight: heights.dashboard.scroll + headerHeight,
    };
  }

  async render(options: ImageRenderOptions): Promise<RenderResponse> {
    let browser: puppeteer.Browser | undefined = undefined;
    let page: puppeteer.Page | undefined = undefined;

    try {
      browser = await this.withTimingMetrics<puppeteer.Browser>(() => {
        this.validateImageOptions(options);
        const launcherOptions = this.getLauncherOptions(options);
        return puppeteer.launch(launcherOptions);
      }, 'launch');

      page = await this.withTimingMetrics<puppeteer.Page>(() => {
        return browser!.newPage();
      }, 'newPage');

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

  private setViewport = async (page: puppeteer.Page, options: ImageRenderOptions): Promise<void> => {
    await page.setViewport({
      width: +options.width,
      height: +options.height,
      deviceScaleFactor: options.deviceScaleFactor ? +options.deviceScaleFactor : 1,
    });
  };

  async takeScreenshot(page: puppeteer.Page, options: ImageRenderOptions): Promise<RenderResponse> {
    try {
      await this.withTimingMetrics(async () => {
        if (this.config.verboseLogging) {
          this.log.debug(
            'Setting viewport for page',
            'width',
            options.width.toString(),
            'height',
            options.height.toString(),
            'deviceScaleFactor',
            options.deviceScaleFactor
          );
        }

        await this.setViewport(page, options);
        await this.preparePage(page, options);
        await this.setTimezone(page, options);

        if (this.config.verboseLogging) {
          this.log.debug('Moving mouse on page', 'x', options.width, 'y', options.height);
        }
        return page.mouse.move(+options.width, +options.height);
      }, 'prepare');

      await this.withTimingMetrics(() => {
        if (this.config.verboseLogging) {
          this.log.debug('Navigating and waiting for all network requests to finish', 'url', options.url);
        }

        return page.goto(options.url, { waitUntil: 'networkidle0', timeout: options.timeout * 1000 });
      }, 'navigate');
    } catch (err) {
      this.log.error('Error while trying to prepare page for screenshot', 'url', options.url, 'err', err.stack);
    }

    let scrollResult: DashboardScrollingResult = {
      scrolled: false,
    };

    if (options.fullPageImage) {
      try {
        scrollResult = await this.withTimingMetrics(() => {
          return this.scrollToLoadAllPanels(page, options);
        }, 'dashboardScrolling');
      } catch (err) {
        this.log.error('Error while scrolling to load all panels', 'url', options.url, 'err', err.stack);
      }
    }

    try {
      await this.withTimingMetrics(() => {
        if (this.config.verboseLogging) {
          this.log.debug('Waiting for dashboard/panel to load', 'timeout', `${options.timeout}s`);
        }

        return page.waitForFunction(
          (isFullPage) => {
            /**
             * panelsRendered value is updated every time that a panel renders. It could happen multiple times in a same panel because scrolling. For full page screenshots
             * we can reach panelsRendered >= panelCount condition even if we have panels that are still loading data and their panelsRenderer value is 0, generating
             * a screenshot with loading panels. It's why the condition for full pages is different from a single panel.
             */
            if (isFullPage) {
              /**
               * data-panelId is the total number of the panels in the dashboard. Rows included.
               * panel-content only exists in non-row panels when the data is loaded.
               * dashboard-row exists only in rows.
               */
              const panelCount = document.querySelectorAll('[data-panelId]').length;
              const panelsRendered = document.querySelectorAll('[class$=\'panel-content\']')
              let panelsRenderedCount = 0
              panelsRendered.forEach((value: Element) => {
                if (value.childElementCount > 0) {
                  panelsRenderedCount++
                }
              })

              const totalPanelsRendered = panelsRenderedCount + document.querySelectorAll('.dashboard-row').length;
              return totalPanelsRendered >= panelCount;
            }

            const panelCount = document.querySelectorAll('.panel').length || document.querySelectorAll('.panel-container').length;
            return (window as any).panelsRendered >= panelCount || (window as any).panelsRendered === undefined;
          },
          {
            timeout: options.timeout * 1000,
          },
          options.fullPageImage || false
        );
      }, 'panelsRendered');
    } catch (err) {
      this.log.error('Error while waiting for the panels to load', 'url', options.url, 'err', err.stack);
    }

    if (!options.filePath) {
      options.filePath = uniqueFilename(os.tmpdir()) + '.png';
    }

    await this.setPageZoomLevel(page, this.config.pageZoomLevel);

    if (this.config.verboseLogging) {
      this.log.debug('Taking screenshot', 'filePath', options.filePath);
    }

    await this.withTimingMetrics(async () => {
      if (scrollResult.scrolled) {
        await this.setViewport(page, {
          ...options,
          height: scrollResult.scrollHeight,
        });
      }
      return page.screenshot({ path: options.filePath, fullPage: options.fullPageImage, captureBeyondViewport: options.fullPageImage || false });
    }, 'screenshot');

    if (options.scaleImage) {
      const scaled = `${options.filePath}_${Date.now()}_scaled.png`;
      const w = +options.width / options.scaleImage;
      const h = +options.height / options.scaleImage;

      await this.withTimingMetrics(async () => {
        const file = await Jimp.read(options.filePath);
        await file
          .resize(w, h)
          // .toFormat('webp', {
          //   quality: 70, // 80 is default
          // })
          .writeAsync(scaled);

        fs.renameSync(scaled, options.filePath);
      }, 'imageResize');
    }

    return { filePath: options.filePath };
  }

  async renderCSV(options: RenderOptions): Promise<RenderCSVResponse> {
    let browser;
    let page: any;

    try {
      this.validateRenderOptions(options);
      const launcherOptions = this.getLauncherOptions(options);
      browser = await puppeteer.launch(launcherOptions);
      page = await browser.newPage();
      this.addPageListeners(page);

      return await this.exportCSV(page, options);
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

  async exportCSV(page: any, options: RenderOptions): Promise<RenderCSVResponse> {
    await this.preparePage(page, options);
    await this.setTimezone(page, options);

    const downloadPath = uniqueFilename(os.tmpdir());
    fs.mkdirSync(downloadPath);
    const watcher = chokidar.watch(downloadPath);
    let downloadFilePath = '';
    watcher.on('add', (file) => {
      if (!file.endsWith('.crdownload')) {
        downloadFilePath = file;
      }
    });

    await page._client().send('Page.setDownloadBehavior', { behavior: 'allow', downloadPath: downloadPath });

    if (this.config.verboseLogging) {
      this.log.debug('Navigating and waiting for all network requests to finish', 'url', options.url);
    }

    await page.goto(options.url, { waitUntil: 'networkidle0', timeout: options.timeout * 1000 });

    if (this.config.verboseLogging) {
      this.log.debug('Waiting for download to end');
    }

    const startDate = Date.now();
    while (Date.now() - startDate <= options.timeout * 1000) {
      if (downloadFilePath !== '') {
        break;
      }
      await new Promise((resolve) => setTimeout(resolve, 500));
    }

    if (downloadFilePath === '') {
      throw new Error(`Timeout exceeded while waiting for download to end`);
    }

    await watcher.close();

    let filePath = downloadFilePath;
    if (options.filePath) {
      fs.copyFileSync(downloadFilePath, options.filePath);
      fs.unlinkSync(downloadFilePath);
      fs.rmdirSync(path.dirname(downloadFilePath));
      filePath = options.filePath;
    }

    return { filePath, fileName: path.basename(downloadFilePath) };
  }

  async withTimingMetrics<T>(callback: () => Promise<T>, step: string): Promise<T> {
    if (this.config.timingMetrics) {
      const endTimer = this.metrics.durationHistogram.startTimer({ step });
      const res = await callback();
      endTimer();

      return res;
    } else {
      return callback();
    }
  }

  addPageListeners(page: puppeteer.Page) {
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

  removePageListeners(page: puppeteer.Page) {
    page.off('error', this.logError);
    page.off('pageerror', this.logPageError);
    page.off('requestfailed', this.logRequestFailed);
    page.off('console', this.logConsoleMessage);

    if (this.config.verboseLogging) {
      page.off('request', this.logRequest);
      page.off('requestfinished', this.logRequestFinished);
      page.off('close', this.logPageClosed);
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
    if (msgType === 'error' && msg.text() !== 'JSHandle@object') {
        this.log.error('Browser console error', 'msg', msg.text(), 'url', loc.url, 'line', loc.lineNumber, 'column', loc.columnNumber);
      return;
    }

    this.log.debug(`Browser console ${msgType}`, 'msg', msg.text(), 'url', loc.url, 'line', loc.lineNumber, 'column', loc.columnNumber);
  };

  logRequest = (req: any) => {
    this.log.debug('Browser request', 'url', req.url(), 'method', req.method());
  };

  logRequestFailed = (req: any) => {
    let failureError = ""
    const failure = req?.failure();
    if (failure) {
      failureError = failure.errorText
    }
    this.log.error('Browser request failed', 'url', req.url(), 'method', req.method(), 'failure', failureError);
  };

  logRequestFinished = (req: any) => {
    this.log.debug('Browser request finished', 'url', req.url(), 'method', req.method());
  };

  logPageClosed = () => {
    this.log.debug('Browser page closed');
  };

  private async setPageZoomLevel(page: puppeteer.Page, zoomLevel: number) {
    if (this.config.verboseLogging) {
      this.log.debug('Setting zoom level', 'zoomLevel', zoomLevel);
    }
    await page.evaluate((zoomLevel: number) => {
      (document.body.style as any).zoom = zoomLevel;
    }, zoomLevel);
  }
}
