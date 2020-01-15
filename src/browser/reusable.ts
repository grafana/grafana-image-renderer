import * as puppeteer from 'puppeteer';
import { Browser, RenderResponse, BrowserTimings } from './browser';
import { Logger } from '../logger';
import { RenderingConfig } from '../config';

export class ReusableBrowser extends Browser {
  browser: puppeteer.Browser;

  constructor(config: RenderingConfig, log: Logger, timings: BrowserTimings) {
    super(config, log, timings);
  }

  async start(): Promise<void> {
    const launcherOptions = this.getLauncherOptions({});

    this.browser = await this.timings.launch(
      async () =>
        // launch browser
        await puppeteer.launch(launcherOptions)
    );
  }

  async render(options): Promise<RenderResponse> {
    let context: puppeteer.BrowserContext;
    let page: puppeteer.Page;

    try {
      this.validateOptions(options);
      context = await this.browser.createIncognitoBrowserContext();

      page = await this.timings.newPage(
        async () =>
          // open a new page
          await context.newPage()
      );

      return await this.takeScreenshot(page, options);
    } finally {
      if (page) {
        await page.close();
      }
      if (context) {
        await context.close();
      }
    }
  }
}
