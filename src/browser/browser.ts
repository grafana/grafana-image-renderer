import * as os from 'os';
import * as uniqueFilename from 'unique-filename';
import * as puppeteer from 'puppeteer';
import * as chokidar from 'chokidar';
import * as path from 'path';
import * as fs from 'fs';
import * as promClient from 'prom-client';
import * as Jimp from 'jimp';
import { Logger } from '../logger';
import { RenderingConfig } from '../config/rendering';
import { HTTPHeaders, ImageRenderOptions, RenderOptions } from '../types';
import { StepTimeoutError } from './error';
import { getPDFOptionsFromURL } from './pdf';
import { trace, Tracer } from '@opentelemetry/api';

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
  tracer: Tracer;

  constructor(protected config: RenderingConfig, protected log: Logger, protected metrics: Metrics) {
    this.log.debug('Browser initialized', 'config', this.config);
    if (config.tracing.url) {
      this.tracer = trace.getTracer('browser');
    }
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

    if (!options.headers['Accept-Language'] && this.config.acceptLanguage) {
      options.headers['Accept-Language'] = this.config.acceptLanguage;
    }

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
      defaultViewport: null,
    };

    if (this.config.chromeBin) {
      launcherOptions.executablePath = this.config.chromeBin;
    }

    launcherOptions.headless = !this.config.headed ? 'shell' : false;

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

    if (options.headers && options.headers['Accept-Language']) {
      const headers = { 'Accept-Language': options.headers['Accept-Language'] };
      this.log.debug(`Setting extra HTTP headers for page`, 'headers', headers);

      await page.setExtraHTTPHeaders(headers as any);
    }

    // automatically accept "Changes you made may not be saved" dialog which could be triggered by saving migrated dashboard schema
    const acceptBeforeUnload = (dialog) => dialog.type() === 'beforeunload' && dialog.accept();
    page.on('dialog', acceptBeforeUnload);
  }

  async scrollToLoadAllPanels(page: puppeteer.Page, options: ImageRenderOptions, signal: AbortSignal): Promise<DashboardScrollingResult> {
    const scrollElementSelector = await page.evaluate(() => {
      const pageScrollbarIDSelector = '#page-scrollbar';
      // the page-scrollbar ID was introduced in Grafana 11.1.0
      // these are selectors that are used to find the page scrollbar in older grafana versions
      // there are several because of the various structural changes made to the page
      // using just [class*="scrollbar-view"] doesn't reliably work as it can match other deeply nested child scrollbars
      // TODO remove these once we are sure that the page-scrollbar ID will always present
      const fallbackSelectors = [
        'main > div > [class*="scrollbar-view"]',
        'main > div > div > [class*="scrollbar-view"]',
        'main > div > div > div > [class*="scrollbar-view"]',
        'main > div > div > div > div > [class*="scrollbar-view"]',
        'main > div > div > div > div > div > [class*="scrollbar-view"]',
      ];
      const pageScrollbarSelector = [pageScrollbarIDSelector, ...fallbackSelectors].join(',');
      const hasPageScrollbar = Boolean(document.querySelector(pageScrollbarSelector));
      return hasPageScrollbar ? pageScrollbarSelector : 'body';
    });
    const scrollDelay = options.scrollDelay ?? 500;

    await page.waitForSelector(scrollElementSelector, { signal });
    const heights: { dashboard?: { scroll: number; client: number }; body: { client: number } } = await page.evaluate((scrollElementSelector) => {
      const body = { client: document.body.clientHeight };
      const scrollableElement = document.querySelector(scrollElementSelector);
      if (!scrollableElement) {
        this.log.debug('no scrollable element detected, returning without scrolling');
        return {
          body,
        };
      }

      return {
        dashboard: { scroll: scrollableElement.scrollHeight, client: scrollableElement.clientHeight },
        body,
      };
    }, scrollElementSelector);

    if (!heights.dashboard) {
      return {
        scrolled: false,
      };
    }

    if (heights.dashboard.scroll <= heights.dashboard.client) {
      this.log.debug(
        'client height greather or equal than scroll height, no scrolling',
        'scrollHeight',
        heights.dashboard.scroll,
        'clientHeight',
        heights.dashboard.client
      );
      return {
        scrolled: false,
      };
    }

    const scrolls = Math.floor(heights.dashboard.scroll / heights.dashboard.client);

    for (let i = 0; i < scrolls; i++) {
      await page.evaluate(
        (scrollByHeight, scrollElementSelector) => {
          scrollElementSelector === 'body'
            ? window.scrollBy(0, scrollByHeight)
            : document.querySelector(scrollElementSelector)?.scrollBy(0, scrollByHeight);
        },
        heights.dashboard.client,
        scrollElementSelector
      );

      await new Promise((executor) => setTimeout(executor, scrollDelay));
    }

    await page.evaluate((scrollElementSelector) => {
      scrollElementSelector === 'body' ? window.scrollTo(0, 0) : document.querySelector(scrollElementSelector)?.scrollTo(0, 0);
    }, scrollElementSelector);

    // Header height will be equal to 0 in Kiosk mode
    const headerHeight = heights.body.client - heights.dashboard.client;
    return {
      scrolled: true,
      scrollHeight: heights.dashboard.scroll + headerHeight,
    };
  }

  async render(options: ImageRenderOptions, signal: AbortSignal): Promise<RenderResponse> {
    let browser: puppeteer.Browser | undefined = undefined;
    let page: puppeteer.Page | undefined = undefined;

    try {
      browser = await this.withMonitoring<puppeteer.Browser>('launch', () => {
        this.validateImageOptions(options);
        const launcherOptions = this.getLauncherOptions(options);
        return puppeteer.launch(launcherOptions);
      });

      page = await this.withMonitoring<puppeteer.Page>('newPage', () => {
        return browser!.newPage();
      });

      await this.addPageListeners(page, options.headers);

      return await this.takeScreenshot(page, options, signal);
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

  async takeScreenshot(page: puppeteer.Page, options: ImageRenderOptions, signal: AbortSignal): Promise<RenderResponse> {
    await this.performStep('prepare', options.url, signal, async () => {
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
    });

    await this.performStep('navigate', options.url, signal, async () => {
      if (this.config.verboseLogging) {
        this.log.debug('Navigating and waiting for all network requests to finish', 'url', options.url);
      }

      await page.goto(options.url, { waitUntil: 'networkidle0', timeout: options.timeout * 1000, signal });
    });

    let scrollResult: DashboardScrollingResult = {
      scrolled: false,
    };

    if (options.fullPageImage) {
      const res = await this.performStep<DashboardScrollingResult>('dashboardScrolling', options.url, signal, () => {
        return this.scrollToLoadAllPanels(page, options, signal);
      });

      if (res) {
        scrollResult = res;
      }
    }

    const isPDF = options.encoding === 'pdf';
    await this.performStep('panelsRendered', options.url, signal, async () => {
      if (this.config.verboseLogging) {
        this.log.debug('Waiting for dashboard/panel to load', 'timeout', `${options.timeout}s`);
      }

      await waitForQueriesAndVisualizations(page, options, signal);
    });

    if (!options.filePath) {
      options.filePath = uniqueFilename(os.tmpdir()) + (isPDF ? '.pdf' : '.png');
    }

    await this.setPageZoomLevel(page, this.config.pageZoomLevel);

    await this.performStep('screenshot', options.url, signal, async () => {
      if (this.config.verboseLogging) {
        this.log.debug('Taking screenshot', 'filePath', options.filePath);
      }

      if (scrollResult.scrolled) {
        await this.setViewport(page, {
          ...options,
          height: scrollResult.scrollHeight,
        });
      }

      if (isPDF) {
        const scale = parseFloat((options.deviceScaleFactor as string) || '1') || 1;
        if (scale < 1) {
          await this.setViewport(page, {
            ...options,
            deviceScaleFactor: 1 / scale,
          });
        }

        const timeoutMs = options.timeout * 1000;
        return page.pdf({
          ...getPDFOptionsFromURL(options.url),
          margin: {
            bottom: 0,
            top: 0,
            right: 0,
            left: 0,
          },
          path: options.filePath,
          scale: 1 / scale,
          timeout: timeoutMs,
        });
      }

      return page.screenshot({ path: options.filePath, fullPage: options.fullPageImage, captureBeyondViewport: false });
    });

    if (options.scaleImage && !isPDF) {
      await this.performStep('imageResize', options.url, signal, async () => {
        const scaled = `${options.filePath}_${Date.now()}_scaled.png`;
        const w = +options.width / options.scaleImage!;
        const h = +options.height / options.scaleImage!;

        const file = await Jimp.read(options.filePath);
        await file
          .resize(w, h)
          // .toFormat('webp', {
          //   quality: 70, // 80 is default
          // })
          .writeAsync(scaled);

        fs.renameSync(scaled, options.filePath);
      });
    }

    return { filePath: options.filePath };
  }

  async renderCSV(options: RenderOptions, signal: AbortSignal): Promise<RenderCSVResponse> {
    let browser;
    let page: any;

    try {
      this.validateRenderOptions(options);
      const launcherOptions = this.getLauncherOptions(options);
      browser = await puppeteer.launch(launcherOptions);
      page = await browser.newPage();

      await this.addPageListeners(page, options.headers);

      return await this.exportCSV(page, options, signal);
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

  async exportCSV(page: any, options: RenderOptions, signal: AbortSignal): Promise<RenderCSVResponse> {
    let downloadFilePath = '';
    let downloadPath = '';
    await this.performStep('prepare', options.url, signal, async () => {
      downloadPath = uniqueFilename(os.tmpdir());

      await this.preparePage(page, options);
      await this.setTimezone(page, options);

      await page._client().send('Page.setDownloadBehavior', { behavior: 'allow', downloadPath: downloadPath });
    });

    fs.mkdirSync(downloadPath);
    const watcher = chokidar.watch(downloadPath);
    watcher.on('add', (file) => {
      if (!file.endsWith('.crdownload')) {
        downloadFilePath = file;
      }
    });

    await this.performStep('navigateCSV', options.url, signal, async () => {
      if (this.config.verboseLogging) {
        this.log.debug('Navigating and waiting for all network requests to finish', 'url', options.url);
      }

      await page.goto(options.url, { waitUntil: 'networkidle0', timeout: options.timeout * 1000, signal });
    });

    await this.performStep('downloadCSV', options.url, signal, async () => {
      if (this.config.verboseLogging) {
        this.log.debug('Waiting for download to end');
      }

      const startDate = Date.now();
      while (Date.now() - startDate <= options.timeout * 1000) {
        if (signal.aborted) {
          this.log.warn('Signal aborted while performing step', 'step', 'downloadCSV', 'url', options.url);
          throw new StepTimeoutError('downloadCSV');
        }
        if (downloadFilePath !== '') {
          break;
        }
        await new Promise((resolve) => setTimeout(resolve, 500));
      }

      if (downloadFilePath === '') {
        throw new StepTimeoutError('downloadCSV');
      }
    });

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

  async performStep<T>(step: string, url: string, signal: AbortSignal, callback: () => Promise<T>): Promise<T | undefined> {
    if (this.config.verboseLogging) {
      this.log.debug('Step begins', 'step', step, 'url', url);
    }

    try {
      const res = await this.withMonitoring(step, callback);

      if (signal.aborted) {
        this.log.warn('Signal aborted while performing step', 'step', step, 'url', url);
        throw new StepTimeoutError(step);
      }

      if (this.config.verboseLogging) {
        this.log.debug('Step ends', 'step', step, 'url', url);
      }

      return res;
    } catch (err) {
      if (!(err instanceof puppeteer.TimeoutError)) {
        this.log.error('Error while performing step', 'step', step, 'url', url, 'err', err.stack);
        throw err;
      }

      this.log.error('Error while performing step', 'step', step, 'url', url, 'err', err.stack);
    }
  }

  async withMonitoring<T>(step: string, callback: () => Promise<T>): Promise<T> {
    // Wrap callback with tracing if enabled (inner layer)
    if (this.tracer) {
      const originalCallback = callback;
      callback = () =>
        this.tracer.startActiveSpan(step, async (span) => {
          try {
            return await originalCallback();
          } finally {
            span.end();
          }
        });
    }

    // Wrap callback with timing metrics if enabled (outer layer)
    if (this.config.timingMetrics) {
      const originalCallback = callback;
      callback = async () => {
        const endTimer = this.metrics.durationHistogram.startTimer({ step });
        try {
          return await originalCallback();
        } finally {
          endTimer();
        }
      };
    }

    return callback();
  }

  async addPageListeners(page: puppeteer.Page, headers?: HTTPHeaders) {
    page.on('error', this.logError);
    page.on('pageerror', this.logPageError);
    page.on('requestfailed', this.logRequestFailed);
    page.on('console', this.logConsoleMessage);

    if (this.config.tracing.url.trim() != '') {
      await page.setRequestInterception(true);

      page.on('request', this.addTracingHeaders(headers));
    }

    if (this.config.verboseLogging) {
      page.on('request', this.logRequest);
      page.on('requestfinished', this.logRequestFinished);
      page.on('close', this.logPageClosed);
      page.on('response', this.logRedirectResponse);
    }
  }

  removePageListeners(page: puppeteer.Page) {
    page.off('error', this.logError);
    page.off('pageerror', this.logPageError);
    page.off('requestfailed', this.logRequestFailed);
    page.off('console', this.logConsoleMessage);

    // page.off('request', ...) does not work so best to remove all listeners for this event
    page.removeAllListeners('request');

    if (this.config.verboseLogging) {
      page.off('requestfinished', this.logRequestFinished);
      page.off('close', this.logPageClosed);
      page.off('response', this.logRedirectResponse);
    }
  }

  addTracingHeaders = (optionsHeaders?: HTTPHeaders) => {
    return (req: puppeteer.HTTPRequest) => {
      if (!optionsHeaders) {
        req.continue();
        return;
      }

      const headers = req.headers();
      const url = req.url();
      const method = req.method();
      const referer = headers['referer'] ?? '';

      try {
        const urlHostname = new URL(url).hostname;
        const refererHostname = referer ? new URL(referer).hostname : '';
        const shouldAddHeaders = req.isNavigationRequest() || urlHostname === refererHostname;
        this.log.debug('Comparing referer and URL hostnames', 'method', method, 'shouldAddHeaders', shouldAddHeaders, 'url', url, 'referer', referer);

        if (shouldAddHeaders) {
          headers['traceparent'] = optionsHeaders['traceparent'] ?? '';
          headers['tracestate'] = optionsHeaders['tracestate'] ?? '';
        }
      } catch (error) {
        this.log.debug('Failed to add tracing headers', 'url', url, 'referer', referer, 'error', error.message);
      }

      req.continue({ headers });
    };
  };

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

  logRequest = (req: puppeteer.HTTPRequest) => {
    this.log.debug('Browser request', 'url', req.url(), 'method', req.method());
  };

  logRedirectResponse = (resp: puppeteer.HTTPResponse) => {
    const status = resp.status();
    if (status >= 300 && status <= 399 && resp.request().resourceType() === 'document') {
      const headers = resp.headers();
      this.log.debug(`Redirect from ${resp.url()} to ${headers['location']}`);
    }
  };

  logRequestFailed = (req: any) => {
    let failureError = '';
    const failure = req?.failure();
    if (failure) {
      failureError = failure.errorText;
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

declare global {
  interface Window {
    __grafanaSceneContext: object;
    __grafanaRunningQueryCount?: number;
  }
}

async function waitForQueriesAndVisualizations(page: puppeteer.Page, options: ImageRenderOptions, signal: AbortSignal) {
  await page.waitForFunction(
    (isFullPage) => {
      if (window.__grafanaSceneContext) {
        return window.__grafanaRunningQueryCount === 0;
      }

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
        const panelsRendered = document.querySelectorAll("[class$='panel-content']");
        let panelsRenderedCount = 0;
        panelsRendered.forEach((value: Element) => {
          if (value.childElementCount > 0) {
            panelsRenderedCount++;
          }
        });

        const rowCount =
          document.querySelectorAll('.dashboard-row').length || document.querySelectorAll("[data-testid='dashboard-row-container']").length;
        const totalPanelsRendered = panelsRenderedCount + rowCount;
        return totalPanelsRendered >= panelCount;
      }

      const panelCount = document.querySelectorAll('.panel-solo').length || document.querySelectorAll("[class$='panel-container']").length;
      return (window as any).panelsRendered >= panelCount || panelCount === 0;
    },
    {
      timeout: options.timeout * 1000,
      polling: 'mutation',
      signal,
    },
    options.fullPageImage
  );

  // Give some more time for rendering to complete
  await new Promise((resolve) => setTimeout(resolve, 50));
}
