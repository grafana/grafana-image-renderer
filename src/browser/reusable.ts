import * as puppeteer from 'puppeteer';
import { ImageRenderOptions, RenderOptions } from '../types';
import { Browser, RenderResponse, RenderCSVResponse, Metrics } from './browser';
import { Logger } from '../logger';
import { RenderingConfig } from '../config/rendering';

export class ReusableBrowser extends Browser {
  browser: puppeteer.Browser;

  constructor(config: RenderingConfig, log: Logger, metrics: Metrics) {
    super(config, log, metrics);
  }

  async start(): Promise<void> {
    const launcherOptions = this.getLauncherOptions({});
    this.browser = await puppeteer.launch(launcherOptions);
  }

  async render(options: ImageRenderOptions, signal: AbortSignal): Promise<RenderResponse> {
    let context: puppeteer.BrowserContext | undefined;
    let page: puppeteer.Page | undefined;

    try {
      page = await this.withMonitoring<puppeteer.Page>('newPage', async () => {
        this.validateImageOptions(options);
        context = await this.browser.createBrowserContext();
        return context.newPage();
      });

      if (options.timezone) {
        // set timezone
        await page.emulateTimezone(options.timezone);
      }

      await this.addPageListeners(page, options.headers);

      return await this.takeScreenshot(page, options, signal);
    } finally {
      if (page) {
        this.removePageListeners(page);
        await page.close();
      }
      if (context) {
        await context.close();
      }
    }
  }

  async renderCSV(options: RenderOptions, signal: AbortSignal): Promise<RenderCSVResponse> {
    let context: puppeteer.BrowserContext | undefined;
    let page: puppeteer.Page | undefined;

    try {
      this.validateRenderOptions(options);
      context = await this.browser.createBrowserContext();
      page = await context.newPage();

      if (options.timezone) {
        // set timezone
        await page.emulateTimezone(options.timezone);
      }

      await this.addPageListeners(page, options.headers);

      return await this.exportCSV(page, options, signal);
    } finally {
      if (page) {
        this.removePageListeners(page);
        await page.close();
      }
      if (context) {
        await context.close();
      }
    }
  }
}
