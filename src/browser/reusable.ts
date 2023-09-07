import * as puppeteer from 'puppeteer';
import { ImageRenderOptions, RenderOptions } from '../types';
import { Browser, RenderResponse, RenderCSVResponse, Metrics } from './browser';
import { Logger } from '../logger';
import { RenderingConfig } from '../config';

export class ReusableBrowser extends Browser {
  browser: puppeteer.Browser;

  constructor(config: RenderingConfig, log: Logger, metrics: Metrics) {
    super(config, log, metrics);
  }

  async start(): Promise<void> {
    const launcherOptions = this.getLauncherOptions({});
    this.browser = await puppeteer.launch(launcherOptions);
  }

  async render(options: ImageRenderOptions): Promise<RenderResponse> {
    let context: puppeteer.BrowserContext | undefined;
    let page: puppeteer.Page | undefined;

    try {
      page = await this.withTimingMetrics<puppeteer.Page>(async () => {
        this.validateImageOptions(options);
        context = await this.browser.createIncognitoBrowserContext();
        return context.newPage();
      }, 'newPage');

      if (options.timezone) {
        // set timezone
        await page.emulateTimezone(options.timezone);
      }

      this.addPageListeners(page);

      return await this.takeScreenshot(page, options);
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

  async renderCSV(options: RenderOptions): Promise<RenderCSVResponse> {
    let context: puppeteer.BrowserContext | undefined;
    let page: puppeteer.Page | undefined;

    try {
      this.validateRenderOptions(options);
      context = await this.browser.createIncognitoBrowserContext();
      page = await context.newPage();

      if (options.timezone) {
        // set timezone
        await page.emulateTimezone(options.timezone);
      }

      this.addPageListeners(page);

      return await this.exportCSV(page, options);
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
