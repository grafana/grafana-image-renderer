import * as puppeteer from 'puppeteer';
import { Browser, RenderResponse, RenderOptions } from './browser';
import { Logger } from '../logger';
import { RenderingConfig } from '../config';
import CancellationToken from 'cancellationtoken';

export class ReusableBrowser extends Browser {
  browser: puppeteer.Browser;

  constructor(config: RenderingConfig, log: Logger) {
    super(config, log);
  }

  async start(): Promise<void> {
    const launcherOptions = this.getLauncherOptions({});
    this.browser = await puppeteer.launch(launcherOptions);
  }

  async render(token: CancellationToken, options: RenderOptions): Promise<RenderResponse> {
    let context: puppeteer.BrowserContext | undefined;
    let page: puppeteer.Page | undefined;

    try {
      this.validateOptions(options);
      context = await this.browser.createIncognitoBrowserContext();
      page = await context.newPage();

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
}
