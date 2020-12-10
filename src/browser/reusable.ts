import * as puppeteer from 'puppeteer';
import { Browser, RenderResponse, RenderOptions } from './browser';
import { Logger } from '../logger';
import { RenderingConfig } from '../config';

export class ReusableBrowser extends Browser {
  browser: puppeteer.Browser;
  page: puppeteer.Page;

  constructor(config: RenderingConfig, log: Logger) {
    super(config, log);
  }

  async start(): Promise<void> {
    const launcherOptions = this.getLauncherOptions({});
    this.browser = await puppeteer.launch(launcherOptions);
  }

  async render(options: RenderOptions): Promise<RenderResponse> {
    let context: puppeteer.BrowserContext | undefined;
    console.log('reusable');

    try {
      this.validateOptions(options);

      if (!this.page) {
        context = await this.browser.createIncognitoBrowserContext();
        console.log('new page');
        this.page = await context.newPage();
        this.addPageListeners(this.page);
      }

      if (options.timezone) {
        // set timezone
        await this.page.emulateTimezone(options.timezone);
      }

      return await this.takeScreenshot(this.page, options);
    } catch (err) {
      if (this.page) {
        this.removePageListeners(this.page);
        await this.page.close();
        this.page = null;
      }

      if (context) {
        await context.close();
      }

      throw err;
    }
  }
}
